package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// exportPollTimeout bounds how long the CLI waits for a server-side export build.
const exportPollTimeout = 5 * time.Minute

// runJobExport downloads a completed job's results via the shared async export
// pipeline: the server builds the file once (streamed into S3, cached) and the
// CLI pulls it from a presigned URL. This replaces the older sync streaming
// download, so large jobs (up to 1M rows) never materialize every row in the CLI.
//
// kind is the export adapter kind ("validity", "identity", "password"); format is
// csv|json|xlsx; filter is the kind-specific row filter ("all", "valid",
// "invalid", "found", "breached"). When out is empty the bytes go to stdout.
func runJobExport(ctx context.Context, client api.API, kind, jobID, out, format, filter string) error {
	spinner := startSpinner("Preparing export...")

	state, err := client.CreateExport(ctx, kind, jobID, format, filter)
	if err != nil {
		stopSpinner(spinner)
		return err
	}

	deadline := time.Now().Add(exportPollTimeout)
	for state.Status != "ready" {
		if state.Status == "failed" {
			stopSpinner(spinner)
			return fmt.Errorf("export failed: %s", firstNonEmpty(state.Error, "please try again"))
		}
		if time.Now().After(deadline) {
			stopSpinner(spinner)
			return fmt.Errorf("export timed out; the job may be very large - please try again shortly")
		}
		select {
		case <-ctx.Done():
			stopSpinner(spinner)
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
		state, err = client.ExportStatus(ctx, kind, jobID, format, filter)
		if err != nil {
			stopSpinner(spinner)
			return err
		}
	}

	// Re-sign a fresh attachment URL (rebuilds transparently if the object aged out).
	url := state.URL
	rows := state.RowCount
	if dl, derr := client.ExportDownloadURL(ctx, kind, jobID, format, filter); derr == nil && dl.URL != "" {
		url = dl.URL
		if dl.RowCount > 0 {
			rows = dl.RowCount
		}
	}
	if url == "" {
		stopSpinner(spinner)
		return fmt.Errorf("export is ready but no download URL was returned")
	}

	blob, err := client.FetchURL(ctx, url)
	stopSpinner(spinner)
	if err != nil {
		return err
	}

	if out == "" {
		fmt.Print(string(blob))
		return nil
	}
	if err := writeFileBytes(out, blob); err != nil {
		return err
	}
	output.SuccessMsg(fmt.Sprintf("Wrote %d %s to %s", rows, plural(rows, "row", "rows"), out))
	return nil
}

// flagString returns a string flag value, or "" when the flag is absent.
func flagString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

// validityExportFilter maps a command's status/only flags to a validity export
// filter ("all", "valid", "invalid", "catch-all"). An explicit --status wins.
func validityExportFilter(cmd *cobra.Command) string {
	if s := strings.ToLower(strings.TrimSpace(flagString(cmd, "status"))); s != "" {
		return s
	}
	validOnly, invalidOnly, _, _ := resolveOnlyFilters(cmd)
	switch {
	case validOnly:
		return "valid"
	case invalidOnly:
		return "invalid"
	default:
		return "all"
	}
}

// nonValidityExportFilter maps identity/password flags to an export filter.
// identity: --found-only / --only found → "found"; password: --breached → "breached".
func nonValidityExportFilter(cmd *cobra.Command, jt string) string {
	_, _, foundOnly, breachedOnly := resolveOnlyFilters(cmd)
	if b, _ := cmd.Flags().GetBool("breached"); b {
		breachedOnly = true
	}
	if f, _ := cmd.Flags().GetBool("found-only"); f {
		foundOnly = true
	}
	switch jt {
	case "identity":
		if foundOnly {
			return "found"
		}
	case "password":
		if breachedOnly {
			return "breached"
		}
	}
	return "all"
}
