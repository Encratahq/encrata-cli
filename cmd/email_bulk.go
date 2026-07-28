package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
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
  encrata email bulk emails.csv --out results.xlsx --columns email,status,trust_grade
  encrata email bulk emails.csv --out results.json --format json
  cat emails.csv | encrata email bulk - --job`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEmailBulk,
}

func init() {
	emailBulkCmd.Flags().Bool("stream", false, "Force live streaming (SSE) mode")
	emailBulkCmd.Flags().Bool("job", false, "Force async job mode")
	emailBulkCmd.Flags().String("out", "", "Write results to a file (.csv, .xlsx, or .json)")
	emailBulkCmd.Flags().String("format", "", "Export format: csv, xlsx, or json (default: inferred from --out)")
	emailBulkCmd.Flags().StringSlice("columns", nil, "Columns to export (email, status, reason always included)")
	emailBulkCmd.Flags().Bool("valid-only", false, "Export only rows whose status is valid")
	emailBulkCmd.Flags().Bool("found-only", false, "Skip rows that carry no enrichment data")
	emailBulkCmd.Flags().StringSlice("fields", nil, "Deprecated alias for --columns")
}

func runEmailBulk(cmd *cobra.Command, args []string) error {
	forceStream, _ := cmd.Flags().GetBool("stream")
	forceJob, _ := cmd.Flags().GetBool("job")
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

	useJob := forceJob || (!forceStream && len(emails) > bulkStreamThreshold)
	if useJob {
		return runBulkJob(cmd, client, fileName, raw, out)
	}
	return runBulkStream(cmd, client, emails, fileName, out, fields)
}

func runBulkStream(cmd *cobra.Command, client *api.Client, emails []string, fileName, out string, fields []string) error {
	total := len(emails)
	asJSON := jsonMode()
	if !asJSON {
		output.Header(fmt.Sprintf("Bulk Validity: %d %s", total, plural(total, "email", "emails")))
	}

	var results []map[string]interface{}
	done := 0
	streamErr := client.BulkValiditySearch(cmd.Context(), emails, fileName, func(ev api.BulkEvent) error {
		switch ev.Type {
		case "result":
			var m map[string]interface{}
			if json.Unmarshal(ev.Data, &m) == nil {
				results = append(results, m)
			}
			done++
			if !asJSON {
				renderProgress(done, total)
			}
		case "error":
			if !asJSON {
				var e map[string]interface{}
				if json.Unmarshal(ev.Data, &e) == nil {
					output.Error(getStr(e, "error"))
				}
			}
		}
		return nil
	})
	if !asJSON {
		fmt.Println()
	}
	if streamErr != nil {
		output.Error(streamErr.Error())
		return streamErr
	}

	if asJSON {
		b, _ := json.Marshal(results)
		output.JSON(b)
		if out != "" {
			return exportBulk(cmd, out, results)
		}
		return nil
	}

	fmt.Println()
	printResultsTable(results, fields)
	printBulkSummaryLine(results)
	if out != "" {
		return exportBulk(cmd, out, results)
	}
	return nil
}

func runBulkJob(cmd *cobra.Command, client *api.Client, fileName string, raw []byte, out string) error {
	spinner := startSpinner("Creating validity job...")
	data, err := client.CreateValidityJobFile(cmd.Context(), fileName, raw)
	stopSpinner(spinner)
	if err != nil {
		output.Error(err.Error())
		return err
	}

	job, err := api.ParseJob(data)
	if err != nil || job.ID == "" {
		output.JSON(data)
		return err
	}

	asJSON := jsonMode()
	if !asJSON {
		output.Header("Validity Job: " + job.ID)
	}

	final, err := client.PollValidityJob(cmd.Context(), job.ID, 2*time.Second, func(j *api.Job) {
		if !asJSON && j.TotalEmails > 0 {
			renderProgress(j.ProcessedCount, j.TotalEmails)
		}
	})
	if !asJSON {
		fmt.Println()
	}
	if err != nil {
		output.Error(err.Error())
		return err
	}

	printJob(final)

	if out != "" {
		// Download the raw result objects and flatten them client-side so job
		// exports use the same column set and formats as streaming exports.
		spinner := startSpinner("Downloading results...")
		blob, err := client.DownloadValidityJob(cmd.Context(), final.ID, "", "json")
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		raw := unwrapArray(json.RawMessage(blob), "results")
		results := make([]map[string]interface{}, 0, len(raw))
		for _, item := range raw {
			if m, ok := item.(map[string]interface{}); ok {
				results = append(results, m)
			}
		}
		return exportBulk(cmd, out, results)
	}
	return nil
}

// normStatus normalizes a validity status for bucketing.
func normStatus(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), "_", "-")
}

// bulkCounts tallies statuses and credits across a result set.
func bulkCounts(results []map[string]interface{}) (total, valid, invalid, catchall, risky, credits int) {
	total = len(results)
	for _, r := range results {
		switch normStatus(field(r, "validity", "status")) {
		case "valid", "deliverable":
			valid++
		case "invalid", "undeliverable":
			invalid++
		case "catch-all", "catchall", "accept-all":
			catchall++
		case "risky", "risk":
			risky++
		}
		credits += intOf(field(r, "credits", "credits_used"))
	}
	if credits == 0 {
		credits = valid
	}
	return
}

// printBulkSummaryLine prints the colored aggregate summary line.
func printBulkSummaryLine(results []map[string]interface{}) {
	total, valid, invalid, catchall, risky, credits := bulkCounts(results)
	fmt.Printf("  Total %d · Valid %s · Invalid %s · Catch-all %s · Risky %s · Credits %d\n",
		total,
		output.Success.Sprintf("%d", valid),
		output.Err.Sprintf("%d", invalid),
		output.Warn.Sprintf("%d", catchall),
		output.Brand.Sprintf("%d", risky),
		credits,
	)
}

// printResultsTable renders Email | Status | Reason plus any --fields columns.
func printResultsTable(results []map[string]interface{}, fields []string) {
	if len(results) == 0 {
		return
	}
	headers := append([]string{"Email", "Status", "Reason"}, fields...)
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		row := []string{
			firstNonEmpty(field(r, "email"), "—"),
			firstNonEmpty(field(r, "validity", "status"), "—"),
			firstNonEmpty(field(r, "reason", "message"), "—"),
		}
		for _, f := range fields {
			row = append(row, cellForField(r, f))
		}
		rows = append(rows, row)
	}
	output.Table(headers, rows)
	fmt.Println()
}

// cellForField renders an arbitrary validity-schema field for table columns.
func cellForField(r map[string]interface{}, f string) string {
	if v, ok := lookupRaw(r, f); ok {
		if arr, ok := v.([]interface{}); ok {
			return firstNonEmpty(joinInterfaces(arr), "—")
		}
	}
	return firstNonEmpty(field(r, f), "—")
}
