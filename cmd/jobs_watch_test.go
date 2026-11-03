package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/config"
	"github.com/Encratahq/cli/internal/notify"
	"github.com/spf13/cobra"
)

// captureNotifications swaps the delivery seam and returns what was "shown".
func captureNotifications(t *testing.T) *[]notify.Message {
	t.Helper()
	var mu sync.Mutex
	got := &[]notify.Message{}
	prev := sendNotification
	sendNotification = func(m notify.Message) error {
		mu.Lock()
		defer mu.Unlock()
		*got = append(*got, m)
		return nil
	}
	t.Cleanup(func() { sendNotification = prev })
	return got
}

func enableNotifications(on ...string) {
	app.cfg.Notifications = config.NotifyPrefs{Enabled: true, On: on}
	app.cfg.AppURL = "https://app.test"
}

func noMinWait(t *testing.T) {
	prev := notifyMinWait
	notifyMinWait = 0
	t.Cleanup(func() { notifyMinWait = prev })
}

// pushFirstFake's first status poll never finishes on its own, so only the
// completion stream's push can end it; the follow-up poll reads the final state.
type pushFirstFake struct {
	*breachesFake
	mu    sync.Mutex
	polls int
}

func (f *pushFirstFake) PollBreachesJob(ctx context.Context, id string, onUpdate func(*api.BreachesJob)) (*api.BreachesJob, error) {
	f.mu.Lock()
	f.polls++
	first := f.polls == 1
	f.mu.Unlock()
	if first {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.breachesFake.PollBreachesJob(ctx, id, onUpdate)
}

func TestBreachesJobEndsOnStreamPush(t *testing.T) {
	f := &pushFirstFake{breachesFake: &breachesFake{
		pushes: []api.JobEvent{
			{Event: api.EventJobCompleted, JobID: "other_job", Kind: "validity"},
			{Event: api.EventJobCompleted, JobID: "bj_1", Kind: "breaches", Processed: 2, Total: 2},
		},
		items: []api.BreachesJobItem{{Email: "user0@acme.com"}, {Email: "user1@acme.com"}},
	}}
	defer withFake(t, f)()
	shown := captureNotifications(t)
	enableNotifications()
	noMinWait(t)

	out := filepath.Join(t.TempDir(), "r.csv")
	cmd := newBreachesTestCmd(t, context.Background(), emailList(2), map[string]string{"job": "true", "out": out})
	done := make(chan error, 1)
	go func() { done <- runBreachesBulk(cmd, "") }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the push did not end the wait")
	}
	if f.polls != 2 {
		t.Fatalf("expected the push to stop the first poll and one final read, got %d polls", f.polls)
	}
	if len(*shown) != 1 || (*shown)[0].Title != "Breach check complete" {
		t.Fatalf("expected one completion notification, got %+v", *shown)
	}
	if want := "https://app.test/email/breaches?job=bj_1"; (*shown)[0].URL != want {
		t.Errorf("results link = %q, want %q", (*shown)[0].URL, want)
	}
}

func TestBreachesJobNotificationRespectsPreferences(t *testing.T) {
	cases := []struct {
		name    string
		prefs   config.NotifyPrefs
		flags   map[string]string
		status  string
		minWait time.Duration
		want    int
	}{
		{"off by default", config.NotifyPrefs{}, nil, "completed", 0, 0},
		{"enabled", config.NotifyPrefs{Enabled: true}, nil, "completed", 0, 1},
		{"--notify overrides off", config.NotifyPrefs{}, map[string]string{"notify": "true"}, "completed", 0, 1},
		{"--no-notify overrides on", config.NotifyPrefs{Enabled: true}, map[string]string{"no-notify": "true"}, "completed", 0, 0},
		{"outcome filtered out", config.NotifyPrefs{Enabled: true, On: []string{"failed"}}, nil, "completed", 0, 0},
		{"failed outcome wanted", config.NotifyPrefs{Enabled: true, On: []string{"failed"}}, nil, "failed", 0, 1},
		{"quick foreground job", config.NotifyPrefs{Enabled: true}, nil, "completed", time.Hour, 0},
		{"--notify skips the min wait", config.NotifyPrefs{}, map[string]string{"notify": "true"}, "completed", time.Hour, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &breachesFake{finalStatus: c.status}
			defer withFake(t, f)()
			shown := captureNotifications(t)
			app.cfg.Notifications = c.prefs
			prev := notifyMinWait
			notifyMinWait = c.minWait
			defer func() { notifyMinWait = prev }()

			flags := map[string]string{"job": "true", "out": filepath.Join(t.TempDir(), "r.csv")}
			for k, v := range c.flags {
				flags[k] = v
			}
			_ = runBreachesBulk(newBreachesTestCmd(t, context.Background(), emailList(2), flags), "")
			if len(*shown) != c.want {
				t.Fatalf("got %d notifications, want %d", len(*shown), c.want)
			}
		})
	}
}

// watchFake drives `jobs watch`: Subscribe replays pushes, the status endpoint
// returns status for every job.
type watchFake struct {
	api.API
	pushes []api.JobEvent
	status string
}

func (f *watchFake) Subscribe(ctx context.Context, _ api.SubscribeHooks, onEvent func(api.JobEvent) error) error {
	for _, ev := range f.pushes {
		if err := onEvent(ev); err != nil {
			if errors.Is(err, api.ErrStopSubscribe) {
				return nil
			}
			return err
		}
	}
	<-ctx.Done()
	return ctx.Err()
}

func (f *watchFake) GetBreachesJob(_ context.Context, id string) (json.RawMessage, error) {
	return json.RawMessage(`{"job":{"id":"` + id + `","status":"` + f.status + `","total_emails":5,"processed_count":5,"breached_count":2}}`), nil
}

func newWatchTestCmd(ctx context.Context, flags map[string]string) *cobra.Command {
	c := &cobra.Command{Use: "watch"}
	c.Flags().String("job", "", "")
	c.Flags().String("type", "", "")
	addNotifyFlags(c)
	for k, v := range flags {
		_ = c.Flags().Set(k, v)
	}
	c.SetContext(ctx)
	return c
}

func runWatch(t *testing.T, cmd *cobra.Command) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- runJobsWatch(cmd, nil) }()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("jobs watch did not return")
		return nil
	}
}

func TestJobsWatchExitCodes(t *testing.T) {
	cases := []struct {
		event   string
		wantErr error
	}{
		{api.EventJobCompleted, nil},
		{api.EventJobFailed, errJobUnsuccessful},
		{api.EventJobCancelled, errJobUnsuccessful},
	}
	for _, c := range cases {
		t.Run(c.event, func(t *testing.T) {
			f := &watchFake{pushes: []api.JobEvent{
				{Event: api.EventJobFailed, JobID: "someone_else", Kind: "breaches"},
				{Event: c.event, JobID: "job_1", Kind: "breaches", Processed: 3, Total: 3},
			}}
			defer withFake(t, f)()
			shown := captureNotifications(t)
			enableNotifications()

			err := runWatch(t, newWatchTestCmd(context.Background(), map[string]string{"job": "job_1"}))
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if len(*shown) != 1 || !strings.Contains((*shown)[0].Body, "job_1") {
				t.Fatalf("expected one notification for job_1, got %+v", *shown)
			}
		})
	}
}

func TestJobsWatchStatusCatchesFinishedJob(t *testing.T) {
	// No push ever arrives (the job finished before we connected): the status
	// check with --type ends the wait.
	f := &watchFake{status: "completed"}
	defer withFake(t, f)()
	captureNotifications(t)
	err := runWatch(t, newWatchTestCmd(context.Background(), map[string]string{"job": "bj_9", "type": "breaches"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJobsWatchFollowEndsCleanlyOnInterrupt(t *testing.T) {
	f := &watchFake{pushes: []api.JobEvent{{Event: api.EventJobCompleted, JobID: "a", Kind: "breaches"}}}
	defer withFake(t, f)()
	captureNotifications(t)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	if err := runWatch(t, newWatchTestCmd(ctx, nil)); err != nil {
		t.Fatalf("Ctrl+C while following should exit cleanly, got %v", err)
	}
}

func TestJobsWatchRejectsTypeWithoutJob(t *testing.T) {
	defer withFake(t, &watchFake{})()
	err := runJobsWatch(newWatchTestCmd(context.Background(), map[string]string{"type": "breaches"}), nil)
	if err == nil || !strings.Contains(err.Error(), "--type needs --job") {
		t.Fatalf("expected a flag error, got %v", err)
	}
}

func TestNotifyBatcherCoalescesBurst(t *testing.T) {
	shown := captureNotifications(t)
	b := newNotifyBatcher(notifyPolicy{enabled: true, on: map[string]bool{"completed": true, "failed": true}, appURL: "https://app.test"})
	b.window = time.Hour // only close() flushes
	b.add(api.JobEvent{Event: api.EventJobCompleted, JobID: "a", Kind: "breaches"})
	b.add(api.JobEvent{Event: api.EventJobFailed, JobID: "b", Kind: "breaches"})
	b.add(api.JobEvent{Event: api.EventJobCancelled, JobID: "c", Kind: "breaches"}) // not wanted
	b.close()
	if len(*shown) != 1 {
		t.Fatalf("expected one coalesced notification, got %d", len(*shown))
	}
	m := (*shown)[0]
	if m.Title != "2 bulk jobs finished" || m.Body != "1 completed · 1 failed" {
		t.Errorf("unexpected batch message: %+v", m)
	}
	if m.URL != "https://app.test/email/breaches?job=a%2Cb" {
		t.Errorf("batch link = %q", m.URL)
	}
}

func TestJobMessage(t *testing.T) {
	ev := api.JobEvent{Event: api.EventJobCompleted, JobID: "bj_1", Kind: "breaches",
		Processed: 1240, Total: 1240, Counts: map[string]int{"breached": 37}}
	m := jobMessage(ev, "https://app.encrata.com/")
	if m.Title != "Breach check complete" {
		t.Errorf("title = %q", m.Title)
	}
	if m.Body != "1,240 checked · 37 breached · 1,203 clean\nJob bj_1" {
		t.Errorf("body = %q", m.Body)
	}
	if m.URL != "https://app.encrata.com/email/breaches?job=bj_1" {
		t.Errorf("url = %q", m.URL)
	}
	if u := jobResultsURL("https://app.encrata.com", "phone", "x"); u != "https://app.encrata.com" {
		t.Errorf("unknown kind should link the dashboard home, got %q", u)
	}
}

func TestParseNotifyEvents(t *testing.T) {
	got, err := parseNotifyEvents([]string{"Failed", "canceled", "failed"})
	if err != nil || strings.Join(got, ",") != "failed,cancelled" {
		t.Fatalf("got %v, %v", got, err)
	}
	if _, err := parseNotifyEvents([]string{"progress"}); err == nil {
		t.Fatal("expected an error for an unknown outcome")
	}
	if all, _ := parseNotifyEvents(nil); len(all) != 3 {
		t.Fatalf("default should be every outcome, got %v", all)
	}
}
