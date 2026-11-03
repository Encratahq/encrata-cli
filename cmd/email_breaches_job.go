package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// runBreachesJob submits the list as an async breaches job, waits for it with a
// live progress bar, then pages the rows back and writes the same result file
// and summary as the stream path. Ctrl+C stops waiting but never cancels the job.
//
// The wait ends on whichever comes first: the completion stream's push for this
// job, or the status poll reaching a terminal state (the fallback when the
// stream is unavailable). A desktop notification follows if the user opted in.
func runBreachesJob(cmd *cobra.Command, client api.API, fileName string, emails []string, out string) error {
	ctx := cmd.Context()
	asJSON := jsonMode()
	start := time.Now()
	policy := resolveNotifyPolicy(cmd)

	watcher := startJobWatcher(ctx, client)
	defer watcher.stop()

	spinner := startSpinner("Creating breaches job...")
	data, err := client.CreateBreachesJob(ctx, emails, fileName)
	stopSpinner(spinner)
	if err != nil {
		output.Error(err.Error())
		return err
	}
	job, err := api.ParseBreachesJob(data)
	if err == nil && job.ID == "" {
		err = fmt.Errorf("the server did not return a job id")
	}
	if err != nil {
		output.Error(err.Error())
		return err
	}

	if !asJSON {
		output.Header("Breaches Job: " + job.ID)
	}
	progress := func(j *api.BreachesJob) {
		if !asJSON && j.TotalEmails > 0 {
			renderProgress(j.ProcessedCount, j.TotalEmails)
		}
	}
	pollCtx, stopPoll := context.WithCancel(ctx)
	defer stopPoll()
	go func() {
		select {
		case <-watcher.finished(job.ID):
			stopPoll()
		case <-pollCtx.Done():
		}
	}()
	final, err := client.PollBreachesJob(pollCtx, job.ID, progress)
	if err != nil && ctx.Err() == nil && pollCtx.Err() != nil {
		// The stream announced the job finished: read its final state now. This
		// returns on the first poll, or keeps polling if the status lags the push.
		final, err = client.PollBreachesJob(ctx, job.ID, progress)
	}
	if !asJSON {
		fmt.Println()
	}
	if err == nil && policy.waitedEnough(time.Since(start)) {
		policy.notifyJob(breachesJobEvent(final))
	}
	if err != nil {
		if ctx.Err() != nil {
			return detachFromJob(job.ID, "Still running in the background. Check it with: encrata jobs status "+job.ID+" --type breaches")
		}
		output.Error(err.Error())
		printJobHint("Check it with: encrata jobs status " + job.ID + " --type breaches")
		return err
	}
	if !breachesJobSucceeded(final.Status) {
		msg := fmt.Sprintf("breaches job %s %s", final.ID, strings.ToLower(final.Status))
		if final.ErrorMessage != "" {
			msg += ": " + final.ErrorMessage
		}
		output.Error(msg)
		return fmt.Errorf("%s", msg)
	}

	results, err := fetchBreachesJobRows(cmd, client, final)
	if err != nil {
		hint := "Download the results with: encrata jobs download " + final.ID + " --type breaches"
		if ctx.Err() != nil {
			return detachFromJob(final.ID, "The job finished. "+hint)
		}
		output.Error(err.Error())
		printJobHint(hint)
		return err
	}

	jsonRows, err := json.Marshal(results)
	if err != nil {
		return err
	}
	return finishBreaches(cmd, out, results, jsonRows, final.ProcessedCount, final.BreachedCount, final.CreditsUsed)
}

// fetchBreachesJobRows pages a completed job's rows back as stream-shaped rows.
// --only breached is applied server-side so only the needed rows are fetched.
func fetchBreachesJobRows(cmd *cobra.Command, client api.API, job *api.BreachesJob) ([]map[string]interface{}, error) {
	asJSON := jsonMode()
	_, _, _, breachedOnly := resolveOnlyFilters(cmd)
	expected := job.ProcessedCount
	if breachedOnly {
		expected = job.BreachedCount
	}

	if !asJSON {
		output.Dim.Println("  Downloading results...")
	}
	rows := make([]map[string]interface{}, 0, expected)
	err := client.FetchBreachesJobItems(cmd.Context(), job.ID, breachedOnly, func(items []api.BreachesJobItem) error {
		for _, it := range items {
			rows = append(rows, breachesJobRow(it))
		}
		if !asJSON {
			renderProgress(len(rows), expected)
		}
		return nil
	})
	if !asJSON {
		fmt.Println()
	}
	return rows, err
}

// breachesJobRow shapes a job result like a bulk-stream row (the report nested
// under breach_info) so breachExportColumns and the JSON export read both the
// same way. Numbers are float64 to match decoded JSON.
func breachesJobRow(it api.BreachesJobItem) map[string]interface{} {
	row := map[string]interface{}{
		"email":        it.Email,
		"breached":     it.Breached,
		"breach_count": float64(it.BreachCount),
	}
	if report, ok := decodeRow(it.Report); ok {
		row["breach_info"] = report
	}
	return row
}

// breachesJobEvent describes a finished breaches job as a completion-stream
// event, so the notification reads the same whether the push or the poll ended
// the wait. The polled job is the authoritative final state.
func breachesJobEvent(j *api.BreachesJob) api.JobEvent {
	ev := api.JobEvent{
		Event:     api.EventJobCompleted,
		JobID:     j.ID,
		Kind:      "breaches",
		Status:    j.Status,
		Processed: j.ProcessedCount,
		Total:     j.TotalEmails,
		Counts:    map[string]int{"breached": j.BreachedCount},
	}
	switch strings.ToLower(strings.TrimSpace(j.Status)) {
	case "cancelled", "canceled":
		ev.Event = api.EventJobCancelled
	default:
		if !breachesJobSucceeded(j.Status) {
			ev.Event = api.EventJobFailed
		}
	}
	return ev
}

// breachesJobSucceeded reports whether a terminal job status means success.
func breachesJobSucceeded(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "complete", "done", "finished":
		return true
	}
	return false
}

// detachFromJob tells the user the job outlives this command (after Ctrl+C) and
// returns errJobDetached. In --json mode the notice is a JSON object on STDOUT.
func detachFromJob(id, hint string) error {
	if jsonMode() {
		payload, _ := json.Marshal(map[string]interface{}{
			"job_id":   id,
			"type":     "breaches",
			"detached": true,
			"message":  hint,
		})
		output.JSON(payload)
		return errJobDetached
	}
	fmt.Fprintf(os.Stderr, "\n  Job %s\n", output.Accent.Sprint(id))
	printJobHint(hint)
	return errJobDetached
}

// printJobHint prints a dim follow-up hint to STDERR.
func printJobHint(hint string) {
	fmt.Fprintln(os.Stderr, "  "+output.Dim.Sprint(hint))
}
