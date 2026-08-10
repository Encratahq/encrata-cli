package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// bulkStreamThreshold is the row count above which bulk validation switches from
// live streaming (SSE) to an async job automatically.
const bulkStreamThreshold = 1000

var emailBulkCmd = &cobra.Command{
	Use:   "bulk [file.csv]",
	Short: "Validate a batch of emails from a file or STDIN",
	Long: `Validate many emails at once. Emails are read from a CSV/text file argument
or from STDIN. Small batches stream live results; large batches (or --job) run as
an async job that is polled to completion.

Examples:
  encrata email bulk emails.csv
  encrata email bulk emails.csv --out results.csv
  encrata email bulk emails.csv --enrich --out results.csv
  encrata email bulk emails.csv --out results.xlsx --columns email,status,trust_grade
  encrata email bulk emails.csv --out results.json --format json
  cat emails.csv | encrata email bulk - --job`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEmailBulk,
}

func init() {
	registerBulkFlags(emailBulkCmd, true)
}

// registerBulkFlags adds the shared bulk execution + export flags to a command.
// includeOut adds --out for commands that don't already provide it.
func registerBulkFlags(cmd *cobra.Command, includeOut bool) {
	cmd.Flags().Bool("stream", false, "Force live streaming (SSE) mode")
	cmd.Flags().Bool("job", false, "Force async job mode")
	cmd.Flags().Bool("enrich", false, "Run the full per-email report so every column is filled (1 credit per email)")
	cmd.Flags().Int("concurrency", 8, "Parallel lookups when --enrich is set")
	cmd.Flags().String("format", "", "Export format: csv, xlsx, or json (default: inferred from --out)")
	cmd.Flags().StringSlice("columns", nil, "Columns to export (email, status, reason always included)")
	cmd.Flags().String("only", "", "Export only rows matching: valid | invalid | found")
	if includeOut {
		cmd.Flags().String("out", "", "Write results to a file (.csv, .xlsx, or .json)")
	}
	// Deprecated compatibility flags (hidden, superseded by --only / --columns).
	cmd.Flags().Bool("valid-only", false, "Deprecated: use --only valid")
	cmd.Flags().Bool("found-only", false, "Deprecated: use --only found")
	cmd.Flags().StringSlice("fields", nil, "Deprecated alias for --columns")
	_ = cmd.Flags().MarkHidden("valid-only")
	_ = cmd.Flags().MarkHidden("found-only")
	_ = cmd.Flags().MarkHidden("fields")
}

// validateOnly checks the --only value against the allowed set for a command.
func validateOnly(cmd *cobra.Command, allowed ...string) error {
	only := strings.ToLower(onlyFlag(cmd))
	if only == "" {
		return nil
	}
	for _, a := range allowed {
		if a == only {
			return nil
		}
	}
	return friendlyFormatError(cmd, fmt.Sprintf("--only must be one of: %s", strings.Join(allowed, ", ")))
}

func runEmailBulk(cmd *cobra.Command, args []string) error {
	if err := validateOnly(cmd, "valid", "invalid", "found"); err != nil {
		return err
	}
	forceStream, _ := cmd.Flags().GetBool("stream")
	forceJob, _ := cmd.Flags().GetBool("job")
	enrich, _ := cmd.Flags().GetBool("enrich")
	if forceStream && forceJob {
		return friendlyFormatError(cmd, "choose either --stream or --job, not both")
	}
	out, _ := cmd.Flags().GetString("out")
	fields, _ := cmd.Flags().GetStringSlice("fields")

	path := ""
	if len(args) == 1 {
		path = args[0]
	}
	fileName, emails, raw, err := loadEmails(cmd, path)
	if err != nil {
		return err
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	if enrich {
		return runBulkEnrich(cmd, client, emails, out, fields)
	}

	useJob := forceJob || (!forceStream && len(emails) > bulkStreamThreshold)
	if useJob {
		return runBulkJob(cmd, client, fileName, raw, out)
	}
	return runBulkStream(cmd, client, emails, fileName, out, fields)
}
