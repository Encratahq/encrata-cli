package cmd

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/config"
	"github.com/Encratahq/cli/internal/notify"
	"github.com/Encratahq/cli/internal/password"
	"github.com/spf13/cobra"
)

var (
	// notifyMinWait skips the notification for a job the user waited on only
	// briefly in the foreground: they are still looking at the terminal.
	notifyMinWait = 30 * time.Second
	// notifyCoalesce batches completions that land close together into one
	// notification while following the stream, so a burst never floods the desktop.
	notifyCoalesce = 3 * time.Second

	// sendNotification is the delivery seam; tests replace it.
	sendNotification = notify.Send
)

// jobKindLabels name each bulk-job kind for humans; jobKindPaths are its
// dashboard page, which reopens a job via ?job=<id>[,<id>...].
var (
	jobKindLabels = map[string]string{
		"breaches":   "Breach check",
		"validity":   "Email validity",
		"identity":   "Identity lookup",
		"gender":     "Gender lookup",
		"compliance": "Compliance check",
		"password":   "Password check",
	}
	jobKindPaths = map[string]string{
		"breaches":   "/email/breaches",
		"validity":   "/email/validity",
		"identity":   "/email/identity",
		"gender":     "/email/gender",
		"compliance": "/email/compliance",
	}
)

// notifyPolicy is the resolved local notification preference for one run.
type notifyPolicy struct {
	enabled bool
	forced  bool            // --notify: the user asked for this run, so skip notifyMinWait
	on      map[string]bool // completed | failed | cancelled
	appURL  string
}

// addNotifyFlags registers the per-run override of the saved preference.
func addNotifyFlags(c *cobra.Command) {
	c.Flags().Bool("notify", false, "Show a desktop notification when the job finishes (overrides config)")
	c.Flags().Bool("no-notify", false, "Never show a desktop notification for this run")
}

// resolveNotifyPolicy merges the saved preference with --notify / --no-notify.
func resolveNotifyPolicy(cmd *cobra.Command) notifyPolicy {
	prefs := app.cfg.Notifications
	p := notifyPolicy{enabled: prefs.Enabled, on: map[string]bool{}, appURL: app.cfg.AppURL}
	on := prefs.On
	if len(on) == 0 {
		on = config.NotifyEvents
	}
	for _, e := range on {
		p.on[strings.ToLower(strings.TrimSpace(e))] = true
	}
	if force, _ := cmd.Flags().GetBool("notify"); force {
		p.enabled, p.forced = true, true
	}
	if off, _ := cmd.Flags().GetBool("no-notify"); off {
		p.enabled, p.forced = false, false
	}
	if p.appURL == "" {
		p.appURL = config.DefaultAppURL
	}
	return p
}

// waitedEnough reports whether a foreground wait of d is long enough to notify:
// past notifyMinWait, or always when the run asked for it with --notify.
func (p notifyPolicy) waitedEnough(d time.Duration) bool {
	return p.forced || d >= notifyMinWait
}

// wants reports whether an outcome event should produce a notification.
func (p notifyPolicy) wants(ev api.JobEvent) bool {
	return p.enabled && p.on[strings.TrimPrefix(ev.Event, "job.")]
}

// notifyJob sends one notification for ev if the policy wants it. A delivery
// failure is reported dimly and never fails the command.
func (p notifyPolicy) notifyJob(ev api.JobEvent) {
	if !p.wants(ev) {
		return
	}
	deliver(jobMessage(ev, p.appURL))
}

func deliver(m notify.Message) {
	if err := sendNotification(m); err != nil {
		printJobHint("Desktop notification failed: " + err.Error())
	}
}

// jobMessage builds the notification for one finished job.
func jobMessage(ev api.JobEvent, appURL string) notify.Message {
	label := jobKindLabel(ev.Kind)
	m := notify.Message{URL: jobResultsURL(appURL, ev.Kind, ev.JobID)}
	switch ev.Event {
	case api.EventJobCompleted:
		m.Title = label + " complete"
		m.Body = jobOutcomeLine(ev)
	case api.EventJobFailed:
		m.Title = label + " failed"
		m.Body = fmt.Sprintf("Stopped after %s of %s. Open the job for details.", password.Commas(ev.Processed), password.Commas(ev.Total))
	default:
		m.Title = label + " cancelled"
		m.Body = fmt.Sprintf("Cancelled after %s of %s.", password.Commas(ev.Processed), password.Commas(ev.Total))
	}
	m.Body += "\nJob " + ev.JobID
	return m
}

// batchMessage summarizes several jobs that finished within one coalesce window.
func batchMessage(evs []api.JobEvent, appURL string) notify.Message {
	if len(evs) == 1 {
		return jobMessage(evs[0], appURL)
	}
	var done, failed, cancelled int
	kinds := map[string]bool{}
	ids := make([]string, 0, len(evs))
	for _, ev := range evs {
		switch ev.Event {
		case api.EventJobCompleted:
			done++
		case api.EventJobFailed:
			failed++
		default:
			cancelled++
		}
		kinds[ev.Kind] = true
		ids = append(ids, ev.JobID)
	}
	parts := []string{}
	for _, p := range []struct {
		n    int
		verb string
	}{{done, "completed"}, {failed, "failed"}, {cancelled, "cancelled"}} {
		if p.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", p.n, p.verb))
		}
	}
	m := notify.Message{
		Title: fmt.Sprintf("%d bulk jobs finished", len(evs)),
		Body:  strings.Join(parts, " · "),
		URL:   strings.TrimRight(appURL, "/"),
	}
	if len(kinds) == 1 {
		m.URL = jobResultsURL(appURL, evs[0].Kind, strings.Join(ids, ","))
	}
	return m
}

// jobOutcomeLine renders "1,240 checked · 37 breached · 1,203 clean".
func jobOutcomeLine(ev api.JobEvent) string {
	verb := "processed"
	if ev.Kind == "breaches" || ev.Kind == "validity" || ev.Kind == "compliance" || ev.Kind == "password" {
		verb = "checked"
	}
	parts := []string{password.Commas(ev.Processed) + " " + verb}
	keys := make([]string, 0, len(ev.Counts))
	for k := range ev.Counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, password.Commas(ev.Counts[k])+" "+strings.ReplaceAll(k, "_", " "))
	}
	if ev.Kind == "breaches" {
		if b, ok := ev.Counts["breached"]; ok && ev.Processed >= b {
			parts = append(parts, password.Commas(ev.Processed-b)+" clean")
		}
	}
	return strings.Join(parts, " · ")
}

func jobKindLabel(kind string) string {
	if l, ok := jobKindLabels[kind]; ok {
		return l
	}
	if kind == "" {
		return "Bulk job"
	}
	return strings.ToUpper(kind[:1]) + kind[1:] + " job"
}

// jobResultsURL deep-links to the dashboard page that shows the job's results,
// falling back to the dashboard home for kinds without a known page.
func jobResultsURL(appURL, kind, jobIDs string) string {
	base := strings.TrimRight(appURL, "/")
	path, ok := jobKindPaths[kind]
	if !ok {
		return base
	}
	return base + path + "?job=" + url.QueryEscape(jobIDs)
}

// notifyBatcher coalesces notifications while following the stream: events
// arriving within notifyCoalesce of the first are sent as one notification.
type notifyBatcher struct {
	policy  notifyPolicy
	window  time.Duration
	mu      sync.Mutex
	pending []api.JobEvent
	timer   *time.Timer
}

func newNotifyBatcher(p notifyPolicy) *notifyBatcher {
	return &notifyBatcher{policy: p, window: notifyCoalesce}
}

// add queues ev if the policy wants it and starts the window on the first one.
func (b *notifyBatcher) add(ev api.JobEvent) {
	if !b.policy.wants(ev) {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pending = append(b.pending, ev)
	if b.timer == nil {
		b.timer = time.AfterFunc(b.window, b.flush)
	}
}

// flush sends whatever is queued. close calls it so nothing is lost on exit.
func (b *notifyBatcher) flush() {
	b.mu.Lock()
	evs := b.pending
	b.pending, b.timer = nil, nil
	b.mu.Unlock()
	if len(evs) > 0 {
		deliver(batchMessage(evs, b.policy.appURL))
	}
}

func (b *notifyBatcher) close() {
	b.mu.Lock()
	if b.timer != nil {
		b.timer.Stop()
	}
	b.mu.Unlock()
	b.flush()
}
