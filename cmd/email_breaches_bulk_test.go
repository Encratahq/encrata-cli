package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Encratahq/cli/internal/api"
	"github.com/spf13/cobra"
)

// breachesFake serves the bulk-breaches stream and job surfaces. Un-overridden
// api.API methods panic via the nil embedded interface.
type breachesFake struct {
	api.API

	streamRows int   // result events to emit before streamErr
	streamErr  error // returned after the rows
	streamed   bool

	created     []string
	pollErr     error
	finalStatus string
	items       []api.BreachesJobItem

	pushes []api.JobEvent // completion-stream frames Subscribe delivers
}

func (f *breachesFake) BulkBreachesSearch(_ context.Context, queries []string, _ string, onEvent func(api.BulkEvent) error) error {
	f.streamed = true
	for i := 0; i < f.streamRows && i < len(queries); i++ {
		row := fmt.Sprintf(`{"email":%q,"breach_info":{"breach_count":%d,"services":[]},"credits":1}`, queries[i], i%2)
		if err := onEvent(api.BulkEvent{Type: "result", Data: json.RawMessage(row)}); err != nil {
			return err
		}
	}
	return f.streamErr
}

func (f *breachesFake) CreateBreachesJob(_ context.Context, emails []string, _ string) (json.RawMessage, error) {
	f.created = emails
	return json.RawMessage(fmt.Sprintf(`{"id":"bj_1","status":"queued","total_emails":%d}`, len(emails))), nil
}

func (f *breachesFake) PollBreachesJob(_ context.Context, id string, onUpdate func(*api.BreachesJob)) (*api.BreachesJob, error) {
	if f.pollErr != nil {
		return nil, f.pollErr
	}
	status := f.finalStatus
	if status == "" {
		status = "completed"
	}
	job := &api.BreachesJob{ID: id, Status: status, TotalEmails: len(f.created),
		ProcessedCount: len(f.created), BreachedCount: 1, CreditsUsed: len(f.created)}
	onUpdate(job)
	return job, nil
}

// Subscribe delivers the configured pushes, then holds the stream open like the
// server does until the caller stops it.
func (f *breachesFake) Subscribe(ctx context.Context, _ api.SubscribeHooks, onEvent func(api.JobEvent) error) error {
	for _, ev := range f.pushes {
		if err := onEvent(ev); err != nil {
			return err
		}
	}
	<-ctx.Done()
	return ctx.Err()
}

func (f *breachesFake) FetchBreachesJobItems(_ context.Context, _ string, _ bool, onPage func([]api.BreachesJobItem) error) error {
	return onPage(f.items)
}

// newBreachesTestCmd builds an `email breaches --bulk` command reading stdin.
func newBreachesTestCmd(t *testing.T, ctx context.Context, stdin string, flags map[string]string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: "breaches"}
	registerBreachesFlags(c)
	c.Flags().String("out", "", "")
	for k, v := range flags {
		if err := c.Flags().Set(k, v); err != nil {
			t.Fatalf("set --%s: %v", k, err)
		}
	}
	c.SetIn(strings.NewReader(stdin))
	c.SetContext(ctx)
	return c
}

func emailList(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "user%d@acme.com\n", i)
	}
	return b.String()
}

// writeEmailFile writes n emails to dir/name and returns its path.
func writeEmailFile(t *testing.T, name string, n int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(emailList(n)), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readLinesOf(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.Split(strings.TrimSpace(string(raw)), "\n")
}

func TestBreachesUseJob(t *testing.T) {
	cases := []struct {
		n                   int
		forceJob, forceStrm bool
		want                bool
	}{
		{150, false, false, false},
		{breachesStreamThreshold, false, false, false},
		{breachesStreamThreshold + 1, false, false, true},
		{8569, false, false, true},
		{8569, false, true, false}, // --stream forces live
		{3, true, false, true},     // --job forces a job
	}
	for _, c := range cases {
		if got := breachesUseJob(c.n, c.forceJob, c.forceStrm); got != c.want {
			t.Errorf("breachesUseJob(%d, job=%v, stream=%v) = %v, want %v", c.n, c.forceJob, c.forceStrm, got, c.want)
		}
	}
}

func TestBreachesRejectsJobWithStream(t *testing.T) {
	f := &breachesFake{}
	defer withFake(t, f)()
	cmd := newBreachesTestCmd(t, context.Background(), emailList(3), map[string]string{"job": "true", "stream": "true"})
	err := runBreachesBulk(cmd, "")
	if err == nil || !strings.Contains(err.Error(), "choose either --stream or --job") {
		t.Fatalf("expected flag conflict error, got %v", err)
	}
	if f.streamed || f.created != nil {
		t.Fatal("nothing should be sent when flags conflict")
	}
}

func TestBreachesSmallListStreams(t *testing.T) {
	f := &breachesFake{streamRows: 3}
	defer withFake(t, f)()
	out := filepath.Join(t.TempDir(), "r.csv")
	cmd := newBreachesTestCmd(t, context.Background(), emailList(3), map[string]string{"out": out})
	if err := runBreachesBulk(cmd, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.streamed || f.created != nil {
		t.Fatalf("expected the stream path (streamed=%v, job=%v)", f.streamed, f.created != nil)
	}
	if lines := readLinesOf(t, out); len(lines) != 4 {
		t.Fatalf("expected header + 3 rows, got %d lines", len(lines))
	}
}

func TestBreachesLargeListRunsAsJob(t *testing.T) {
	f := &breachesFake{items: []api.BreachesJobItem{
		{Email: "user0@acme.com", Breached: true, BreachCount: 2,
			Report: json.RawMessage(`{"breach_count":2,"services":[{"name":"Apollo"},{"name":"Dubsmash"}]}`)},
		{Email: "user1@acme.com"},
	}}
	defer withFake(t, f)()
	out := filepath.Join(t.TempDir(), "r.csv")
	n := breachesStreamThreshold + 1
	cmd := newBreachesTestCmd(t, context.Background(), "", map[string]string{"out": out})
	if err := runBreachesBulk(cmd, writeEmailFile(t, "big.csv", n)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.streamed {
		t.Fatal("a list above the threshold must not stream")
	}
	if len(f.created) != n {
		t.Fatalf("expected a job with %d emails, got %d", n, len(f.created))
	}
	lines := readLinesOf(t, out)
	if len(lines) != 3 || lines[0] != "email,breached,breach_count,breaches" {
		t.Fatalf("unexpected result file: %q", lines)
	}
	if lines[1] != "user0@acme.com,yes,2,Apollo | Dubsmash" {
		t.Errorf("breached row = %q", lines[1])
	}
	if lines[2] != "user1@acme.com,no,0," {
		t.Errorf("clean row = %q", lines[2])
	}
}

func TestBreachesForcedJobForSmallList(t *testing.T) {
	f := &breachesFake{}
	defer withFake(t, f)()
	out := filepath.Join(t.TempDir(), "r.json")
	cmd := newBreachesTestCmd(t, context.Background(), emailList(2), map[string]string{"job": "true", "out": out})
	if err := runBreachesBulk(cmd, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.streamed || len(f.created) != 2 {
		t.Fatalf("--job must force a job (streamed=%v, created=%d)", f.streamed, len(f.created))
	}
}

func TestBreachesFailedJobReturnsError(t *testing.T) {
	f := &breachesFake{finalStatus: "failed"}
	defer withFake(t, f)()
	cmd := newBreachesTestCmd(t, context.Background(), emailList(2), map[string]string{"job": "true"})
	if err := runBreachesBulk(cmd, ""); err == nil || !strings.Contains(err.Error(), "failed") {
		t.Fatalf("expected a failed-job error, got %v", err)
	}
}

func TestBreachesJobDetachesOnCtrlC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &breachesFake{pollErr: context.Canceled}
	defer withFake(t, f)()
	cmd := newBreachesTestCmd(t, ctx, emailList(2), map[string]string{"job": "true"})
	err := runBreachesBulk(cmd, "")
	if !errors.Is(err, errJobDetached) {
		t.Fatalf("expected errJobDetached, got %v", err)
	}
}

func TestBreachesStreamCutSavesPartialResults(t *testing.T) {
	f := &breachesFake{streamRows: 3, streamErr: &api.StreamCutError{Err: io.ErrUnexpectedEOF}}
	defer withFake(t, f)()
	out := filepath.Join(t.TempDir(), "partial.csv")
	cmd := newBreachesTestCmd(t, context.Background(), "", map[string]string{"stream": "true", "out": out})

	list := writeEmailFile(t, "list.csv", 5)
	err := runBreachesBulk(cmd, list)
	if err == nil {
		t.Fatal("a cut stream must exit non-zero")
	}
	for _, want := range []string{"The connection closed after", "3 of 5 checked", list + " --bulk --job"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q is missing %q", err.Error(), want)
		}
	}
	if strings.Contains(err.Error(), "unexpected EOF") {
		t.Errorf("raw transport error leaked into the message: %q", err.Error())
	}
	if lines := readLinesOf(t, out); len(lines) != 4 {
		t.Fatalf("expected header + 3 partial rows, got %d lines", len(lines))
	}
}

func TestFriendlyStreamError(t *testing.T) {
	cut := streamFailure{
		err:     &api.StreamCutError{Err: errors.New("stream error: stream ID 1; INTERNAL_ERROR; received from peer")},
		elapsed: 120 * time.Second,
		done:    455,
		total:   8569,
		retry:   "encrata email breaches big.txt --bulk --job",
	}
	got := friendlyStreamError(cut).Error()
	for _, want := range []string{"The connection closed after 2m 00s (455 of 8,569 checked).",
		"Large lists run reliably as a background job:", "encrata email breaches big.txt --bulk --job"} {
		if !strings.Contains(got, want) {
			t.Errorf("message %q is missing %q", got, want)
		}
	}
	if !errors.As(friendlyStreamError(cut), new(*api.StreamCutError)) {
		t.Error("friendly error must still unwrap to the cause")
	}

	// Errors other than a cut stream pass through untouched.
	credits := &api.Error{StatusCode: 402, Message: "Insufficient credits. Need 10, have 2."}
	for _, err := range []error{credits, context.Canceled} {
		if got := friendlyStreamError(streamFailure{err: err}); got != err {
			t.Errorf("friendlyStreamError(%v) = %v, want passthrough", err, got)
		}
	}
}

func TestFormatHelpers(t *testing.T) {
	for n, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 8569: "8,569", 1234567: "1,234,567"} {
		if got := formatCount(n); got != want {
			t.Errorf("formatCount(%d) = %q, want %q", n, got, want)
		}
	}
	if got := formatElapsed(45 * time.Second); got != "45s" {
		t.Errorf("formatElapsed(45s) = %q", got)
	}
	if got := formatElapsed(125 * time.Second); got != "2m 05s" {
		t.Errorf("formatElapsed(125s) = %q", got)
	}
}
