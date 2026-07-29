package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
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

// runBulkEnrich runs the FULL single-email validity report for every address
// concurrently, so exported rows carry every column (confidence, provider, mx,
// domain_trust, smtp, footprint, ...) instead of just status/reason. Costs 1
// credit per email.
func runBulkEnrich(cmd *cobra.Command, client *api.Client, emails []string, out string, fields []string) error {
	total := len(emails)
	asJSON := jsonMode()
	if !asJSON {
		output.Header(fmt.Sprintf("Bulk Enrich: %d %s", total, plural(total, "email", "emails")))
	}

	concurrency, _ := cmd.Flags().GetInt("concurrency")
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > total && total > 0 {
		concurrency = total
	}

	results := make([]map[string]interface{}, total)
	rawResults := make([]json.RawMessage, total)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var done int32
	var mu sync.Mutex

	for i, email := range emails {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, addr string) {
			defer wg.Done()
			defer func() { <-sem }()

			data, err := client.EmailValidity(cmd.Context(), addr)
			row := map[string]interface{}{"email": addr}
			if err != nil {
				row["status"] = "error"
				row["reason"] = err.Error()
			} else if m, ok := decodeRow(data); ok {
				row = m
				if _, hasEmail := row["email"]; !hasEmail {
					row["email"] = addr
				}
				copyRaw := make(json.RawMessage, len(data))
				copy(copyRaw, data)
				rawResults[idx] = copyRaw
			}
			results[idx] = row

			if !asJSON {
				mu.Lock()
				done++
				renderProgress(int(done), total)
				mu.Unlock()
			}
		}(i, email)
	}
	wg.Wait()

	// Drop rows that failed to decode into a payload (nil raw) for the JSON array.
	cleanRaw := make([]json.RawMessage, 0, total)
	for _, raw := range rawResults {
		if len(raw) > 0 {
			cleanRaw = append(cleanRaw, raw)
		}
	}

	if asJSON {
		output.JSON(rawResultsArray(cleanRaw))
		if out != "" {
			return exportBulk(cmd, out, results)
		}
		return nil
	}

	fmt.Println()
	fmt.Println()
	printBulkSummaryLine(results)
	if out != "" {
		return exportBulk(cmd, out, results)
	}
	return nil
}

// decodeRow unmarshals a single JSON object into a map.
func decodeRow(data json.RawMessage) (map[string]interface{}, bool) {
	var m map[string]interface{}
	if json.Unmarshal(data, &m) != nil {
		return nil, false
	}
	return m, true
}

func runBulkStream(cmd *cobra.Command, client *api.Client, emails []string, fileName, out string, fields []string) error {
	total := len(emails)
	asJSON := jsonMode()
	if !asJSON {
		output.Header(fmt.Sprintf("Bulk Validity: %d %s", total, plural(total, "email", "emails")))
	}

	var results []map[string]interface{}
	var rawResults []json.RawMessage
	done := 0
	streamErr := client.BulkValiditySearch(cmd.Context(), emails, fileName, func(ev api.BulkEvent) error {
		switch ev.Type {
		case "result":
			var m map[string]interface{}
			if json.Unmarshal(ev.Data, &m) == nil {
				results = append(results, m)
				raw := make(json.RawMessage, len(ev.Data))
				copy(raw, ev.Data)
				rawResults = append(rawResults, raw)
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
		output.JSON(rawResultsArray(rawResults))
		if out != "" {
			return exportBulk(cmd, out, results)
		}
		return nil
	}

	fmt.Println()
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

// rawResultsArray assembles the raw per-row payloads into a single JSON array,
// preserving the exact server bytes for --json parity with the MCP server.
func rawResultsArray(rows []json.RawMessage) json.RawMessage {
	if len(rows) == 0 {
		return json.RawMessage("[]")
	}
	var buf strings.Builder
	buf.WriteByte('[')
	for i, row := range rows {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(row)
	}
	buf.WriteByte(']')
	return json.RawMessage(buf.String())
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
