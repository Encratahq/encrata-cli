package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// errJobDetached means the user pressed Ctrl+C while waiting on an async job.
// The job keeps running server-side; the follow-up hint is already printed, so
// Execute() exits non-zero without an "Aborted." banner.
var errJobDetached = errors.New("stopped waiting for the job")

// friendlyError replaces an error's message while keeping it unwrappable, so
// exit-code mapping (e.g. *api.Error 402) still works.
type friendlyError struct {
	msg   string
	cause error
}

func (e *friendlyError) Error() string { return e.msg }
func (e *friendlyError) Unwrap() error { return e.cause }

// streamFailure describes a bulk SSE stream that ended in an error.
type streamFailure struct {
	err     error
	elapsed time.Duration
	done    int    // result events received
	total   int    // emails sent
	retry   string // command that runs the same list as an async job
}

// friendlyStreamError maps a dropped stream (proxy reset, unexpected EOF,
// client timeout mid-body) onto an actionable message. Other errors pass
// through unchanged.
func friendlyStreamError(f streamFailure) error {
	var cut *api.StreamCutError
	if !errors.As(f.err, &cut) {
		return f.err
	}
	return &friendlyError{
		msg: fmt.Sprintf("The connection closed after %s (%s of %s checked).\n"+
			"    Large lists run reliably as a background job:\n"+
			"    %s",
			formatElapsed(f.elapsed), formatCount(f.done), formatCount(f.total), f.retry),
		cause: f.err,
	}
}

// handleStreamFailure reports a failed bulk stream without losing the rows that
// already arrived: they are written to out via save (or, in --json mode, emitted
// on STDOUT), then the friendly error is returned so the command exits non-zero.
func handleStreamFailure(f streamFailure, out string, raw []json.RawMessage, save func(out string) error) error {
	friendly := friendlyStreamError(f)

	if jsonMode() {
		payload, _ := json.Marshal(map[string]interface{}{
			"error":           friendly.Error(),
			"detail":          f.err.Error(),
			"partial":         len(raw) > 0,
			"checked":         f.done,
			"total":           f.total,
			"elapsed_seconds": int(f.elapsed.Seconds()),
			"results":         rawResultsArray(raw),
		})
		output.JSON(payload)
	}

	if len(raw) > 0 && out != "" {
		if err := save(out); err != nil {
			output.Error("could not save partial results: " + err.Error())
		} else {
			abs, absErr := filepath.Abs(out)
			if absErr != nil {
				abs = out
			}
			fmt.Fprintf(os.Stderr, "  Saved %s of %s results to %s (partial).\n",
				formatCount(len(raw)), formatCount(f.total), abs)
		}
	}

	if errors.Is(f.err, context.Canceled) {
		return f.err // Ctrl+C: Execute() prints "Aborted."
	}
	output.Error(friendly.Error())
	return friendly
}

// asyncRetryHint builds the command that re-runs a bulk list as a job.
func asyncRetryHint(cmd *cobra.Command, path string, extra ...string) string {
	file := path
	if file == "" || file == "-" {
		file = "<file>"
	}
	hint := cmd.CommandPath() + " " + file
	for _, e := range extra {
		hint += " " + e
	}
	return hint
}

// formatCount renders n with thousands separators (8569 -> "8,569").
func formatCount(n int) string {
	s := strconv.Itoa(n)
	neg := n < 0
	if neg {
		s = s[1:]
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

// formatElapsed renders a duration as "45s" or "2m 05s".
func formatElapsed(d time.Duration) string {
	secs := int(d.Round(time.Second).Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	return fmt.Sprintf("%dm %02ds", secs/60, secs%60)
}
