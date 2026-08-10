package cmd

import (
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Manage async jobs (validity, identity, password)",
}

// printJob renders a job status block with counts, credits, and created time.
func printJob(job *api.Job) {
	status := job.Status
	if colored := statusColor(job.Status); colored != "" {
		status = colored
	}
	output.KV(
		"ID", job.ID,
		"Status", status,
		"Total", fmt.Sprintf("%d", job.TotalEmails),
		"Processed", fmt.Sprintf("%d", job.ProcessedCount),
		"Valid", fmt.Sprintf("%d", job.SuccessCount),
		"Errors", fmt.Sprintf("%d", job.ErrorCount),
		"Credits used", fmt.Sprintf("%d", job.CreditsUsed),
		"Created", jobCreated(job.CreatedAt),
	)
}

// jobCreated renders a job timestamp in local, human-readable form.
func jobCreated(ts string) string {
	if ts == "" {
		return ""
	}
	return timeField(map[string]interface{}{"t": ts}, "t")
}

// coloredStatus returns a color-themed status label, falling back to the raw
// string when no theme is defined for it.
func coloredStatus(s string) string {
	if c := statusColor(s); c != "" {
		return c
	}
	return s
}

func filterRowsByStatus(rows []map[string]interface{}, status string) []map[string]interface{} {
	want := strings.ToLower(strings.TrimSpace(status))
	if want == "" {
		return rows
	}
	filtered := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		raw := strings.ToLower(strings.TrimSpace(firstNonEmpty(getStr(row, "status"), getStr(row, "validity"))))
		normalized := normalizedValidityStatus(raw)
		if raw == want || normalized == want {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func init() {
	// --type selects the job kind on every unified command (default validity).
	for _, c := range []*cobra.Command{
		jobsCreateCmd, jobsListCmd, jobsStatusCmd, jobsResultsCmd,
		jobsDownloadCmd, jobsCancelCmd, jobsRetryCmd,
	} {
		c.Flags().String("type", "validity", "Job type: validity | identity | password")
	}

	// create: identity/validity read emails from the file arg or STDIN;
	// password reads hashes from these flags.
	jobsCreateCmd.Flags().StringSlice("sha1s", nil, "Password: inline SHA-1 hashes (40-char hex)")
	jobsCreateCmd.Flags().String("sha1-file", "", "Password: read SHA-1 hashes from a file")
	jobsCreateCmd.Flags().String("password-file", "", "Password: read plaintext passwords from a file (hashed locally)")
	jobsCreateCmd.Flags().String("file-name", "", "Optional display name for the job")

	jobsResultsCmd.Flags().String("status", "", "Filter results by per-row status (e.g. valid, invalid)")
	jobsResultsCmd.Flags().Int("page", 1, "Result page to fetch")
	jobsResultsCmd.Flags().Int("page-size", 50, "Results per page (identity/password)")
	jobsResultsCmd.Flags().Bool("found-only", false, "Identity: only rows with enrichment data")
	jobsResultsCmd.Flags().Bool("breached", false, "Password: only breached rows")
	jobsResultsCmd.Flags().StringSlice("fields", nil, "Extra columns from the validity schema (e.g. provider,mx)")

	jobsDownloadCmd.Flags().String("format", "csv", "Download format: csv, xlsx, or json (validity)")
	jobsDownloadCmd.Flags().String("status", "", "Filter rows by status")
	jobsDownloadCmd.Flags().Bool("valid-only", false, "Download only rows whose status is valid")
	jobsDownloadCmd.Flags().Bool("found-only", false, "Identity: only rows with enrichment data")
	jobsDownloadCmd.Flags().Bool("breached", false, "Password: only breached rows")
	jobsDownloadCmd.Flags().String("out", "", "Write to a file instead of stdout")

	jobsCmd.AddCommand(
		jobsCreateCmd,
		jobsListCmd,
		jobsStatusCmd,
		jobsResultsCmd,
		jobsDownloadCmd,
		jobsCancelCmd,
		jobsRetryCmd,
	)
}
