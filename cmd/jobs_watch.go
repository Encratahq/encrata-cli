package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// errJobUnsuccessful ends `jobs watch --job` with a non-zero exit when the job
// failed or was cancelled. The outcome is already printed, so no error banner.
var errJobUnsuccessful = errors.New("job did not complete")

// watchStatusEvery is the fallback status check for `jobs watch --job --type`:
// it catches a job that finished before the stream connected (the server only
// pushes live events) or a deployment where the stream is disabled.
const watchStatusEvery = 30 * time.Second

var jobsWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Wait for bulk jobs to finish, live (no polling)",
	Long: `Hold the live completion stream open and print each bulk job as it
finishes (completed, failed, or cancelled). Progress is never streamed.

With --job, wait for that one job and exit: code 0 if it completed, 1 if it
failed or was cancelled. Without --job, follow every job until Ctrl+C.

The stream covers every bulk job you own, including jobs run in other
workspaces, so each line shows the job id and kind.`,
	Example: `  encrata jobs watch
  encrata jobs watch --job job_abc123 && encrata jobs download job_abc123 --type breaches
  encrata jobs watch --job job_abc123 --type breaches --notify
  encrata jobs watch --json`,
	Args: cobra.NoArgs,
	RunE: runJobsWatch,
}

func init() {
	jobsWatchCmd.Flags().String("job", "", "Exit when this job finishes (0 = completed, 1 = failed/cancelled)")
	jobsWatchCmd.Flags().String("type", "", "Job type of --job, enabling a status check for jobs that already finished: validity | identity | password | breaches")
	addNotifyFlags(jobsWatchCmd)
	jobsCmd.AddCommand(jobsWatchCmd)
}

func runJobsWatch(cmd *cobra.Command, _ []string) error {
	jobID, _ := cmd.Flags().GetString("job")
	jobID = strings.TrimSpace(jobID)
	jt, _ := cmd.Flags().GetString("type")
	jt = strings.ToLower(strings.TrimSpace(jt))
	if jt != "" && jobID == "" {
		return friendlyFormatError(cmd, "--type needs --job")
	}
	if jt != "" && jt != "validity" {
		if _, ok := jobStrategies[jt]; !ok {
			return friendlyFormatError(cmd, fmt.Sprintf("unknown --type %q; use validity, identity, password, or breaches", jt))
		}
	}
	client, err := newClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Following: completions are coalesced so a burst is one notification.
	policy := resolveNotifyPolicy(cmd)
	batch := newNotifyBatcher(policy)
	defer batch.close()

	if !jsonMode() {
		if jobID != "" {
			fmt.Fprintf(os.Stderr, "  Waiting for job %s to finish (Ctrl+C to stop)...\n", output.Accent.Sprint(jobID))
		} else {
			fmt.Fprintln(os.Stderr, "  Watching for finished bulk jobs (Ctrl+C to stop)...")
		}
	}

	// finish records the one job --job waits for. It runs on whichever goroutine
	// saw the outcome first; the main goroutine reads outcome after both stop.
	var outcome *api.JobEvent
	finish := func(ev api.JobEvent) error {
		outcome = &ev
		return api.ErrStopSubscribe
	}

	// The status check runs beside the stream and ends the wait the same way.
	statusDone := make(chan api.JobEvent, 1)
	if jobID != "" && jt != "" {
		go watchJobStatus(ctx, client, jt, jobID, statusDone)
	}

	connectedOnce := false
	hooks := api.SubscribeHooks{
		OnConnect: func() {
			if connectedOnce && !jsonMode() {
				printJobHint("Reconnected.")
			}
			connectedOnce = true
		},
		OnRetry: func(err error, wait time.Duration) {
			if !jsonMode() {
				printJobHint(fmt.Sprintf("Stream dropped (%v); reconnecting in %s...", err, wait.Round(time.Second)))
			}
		},
	}

	subDone := make(chan error, 1)
	go func() {
		subDone <- client.Subscribe(ctx, hooks, func(ev api.JobEvent) error {
			if jobID == "" {
				printWatchEvent(ev)
				batch.add(ev)
				return nil
			}
			if ev.JobID != jobID {
				return nil
			}
			return finish(ev)
		})
	}()

	select {
	case err = <-subDone:
	case ev := <-statusDone:
		cancel()
		<-subDone
		if outcome == nil { // the stream did not also deliver it
			_ = finish(ev)
		}
	}

	if jobID == "" {
		if errors.Is(err, context.Canceled) {
			return nil // Ctrl+C is how a follow ends
		}
		return err
	}
	if outcome == nil {
		return err // Ctrl+C before the job finished, or the stream was refused
	}
	printWatchEvent(*outcome)
	policy.notifyJob(*outcome)
	if !outcome.Succeeded() {
		return errJobUnsuccessful
	}
	return nil
}

// watchJobStatus checks jobID's status now and every watchStatusEvery, sending
// its outcome once it is terminal. Errors are ignored: the stream is primary.
func watchJobStatus(ctx context.Context, client api.API, jt, jobID string, out chan<- api.JobEvent) {
	for {
		if ev, ok := jobStatusEvent(ctx, client, jt, jobID); ok {
			out <- ev
			return
		}
		if sleepOrDone(ctx, watchStatusEvery) != nil {
			return
		}
	}
}

// jobStatusEvent fetches one job's status and, if terminal, describes it as a
// completion-stream event.
func jobStatusEvent(ctx context.Context, client api.API, jt, jobID string) (api.JobEvent, bool) {
	var data json.RawMessage
	var err error
	if jt == "validity" {
		data, err = client.GetValidityJob(ctx, jobID)
	} else {
		data, err = jobStrategies[jt].status(ctx, client, jobID)
	}
	if err != nil {
		return api.JobEvent{}, false
	}
	job, err := api.ParseBreachesJob(data) // shared id/status/count field names
	if err != nil || !api.JobIsTerminal(job.Status) {
		return api.JobEvent{}, false
	}
	job.ID = jobID
	ev := breachesJobEvent(job)
	ev.Kind = jt
	if jt != "breaches" {
		ev.Counts = nil
	}
	return ev, true
}

func sleepOrDone(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// printWatchEvent prints one finished job: a JSON line in --json mode,
// otherwise a status-colored summary line.
func printWatchEvent(ev api.JobEvent) {
	if jsonMode() {
		data, _ := json.Marshal(ev)
		fmt.Println(string(data))
		return
	}
	mark, status := output.Success.Sprint("✓"), output.Success.Sprint("completed")
	switch ev.Event {
	case api.EventJobFailed:
		mark, status = output.Err.Sprint("✗"), output.Err.Sprint("failed")
	case api.EventJobCancelled:
		mark, status = output.Dim.Sprint("–"), output.Dim.Sprint("cancelled")
	}
	fmt.Printf("  %s %s  %s  %s  %s\n", mark, output.Accent.Sprint(ev.JobID), ev.Kind, status, jobOutcomeLine(ev))
}
