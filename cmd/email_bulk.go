package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var emailBulkCmd = &cobra.Command{
	Use:   "bulk [file.csv]",
	Short: "Validate a batch of emails from a file or STDIN",
	Long: `Validate many emails at once. Emails are read from a CSV/text file argument
or from STDIN. Every batch runs as an async job, the same one the dashboard
runs, polled here to completion, so counts, credits and the downloaded report
match the dashboard for the same file.

By default only email,status,reason are returned. Add --enrich to fill every
export column (provider, mx, trust grade, ...). Results print to the terminal;
pass --out to also save them to a .csv/.xlsx/.json file.

Use the ` + "`jobs`" + ` command instead when you want to start a job and come back
for it later.

Examples:
  encrata email bulk emails.csv
  encrata email bulk emails.csv --out results.csv
  encrata email bulk emails.csv --enrich --out results.csv
  encrata email bulk emails.csv --out results.xlsx --columns email,status,trust_grade
  encrata email bulk emails.csv --out results.json --format json
  cat emails.csv | encrata email bulk -`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEmailBulk,
}

func init() {
	registerBulkFlags(emailBulkCmd, true)
}

// registerBulkFlags adds the shared bulk execution + export flags to a command.
// includeOut adds --out for commands that don't already provide it.
func registerBulkFlags(cmd *cobra.Command, includeOut bool) {
	// --stream and --job are accepted and ignored so existing scripts keep working:
	// every list now runs as a job.
	cmd.Flags().Bool("stream", false, "Deprecated: ignored, every list runs as a job")
	cmd.Flags().Bool("job", false, "Deprecated: ignored, every list runs as a job")
	cmd.Flags().Bool("enrich", false, "Fill every export column with the full per-email report (same 1 credit/email, slower)")
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
	_ = cmd.Flags().MarkHidden("stream")
	_ = cmd.Flags().MarkHidden("job")
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
	enrich, _ := cmd.Flags().GetBool("enrich")
	fields, _ := cmd.Flags().GetStringSlice("fields")

	path := ""
	if len(args) == 1 {
		path = args[0]
	}
	fileName, emails, raw, err := loadEmails(cmd, path)
	if err != nil {
		return err
	}
	// No --out: auto-name after the uploaded file under encrata-cli-results/.
	out := resolveResultOut(cmd, fileName)

	client, err := newClient()
	if err != nil {
		return err
	}

	if enrich {
		return runBulkEnrich(cmd, client, emails, out, fields)
	}

	// Every list, whatever its size, is a validity job on POST /api/cli/jobs/bulk:
	// the dashboard's path, so the same file gives the same engine, counts, report
	// and credits, and an identical list reuses the existing job instead of being
	// checked and charged twice (ENC-2783).
	return runBulkJob(cmd, client, fileName, raw, out)
}
