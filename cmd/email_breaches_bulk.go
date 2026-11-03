package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// breachesStreamThreshold is the row count above which bulk breach checks switch
// from live streaming (SSE) to an async job automatically. It is far below
// validity's 1000 because breach lookups run at only ~2-4 emails/s, and the
// server cuts any single stream at 120s (Envoy route timeout + cli proxy), so a
// stream must comfortably finish inside that window.
const breachesStreamThreshold = 200

// breachesUseJob decides whether a bulk breach check runs as an async job.
func breachesUseJob(count int, forceJob, forceStream bool) bool {
	return forceJob || (!forceStream && count > breachesStreamThreshold)
}

// runBreachesBulk checks a file/STDIN list of emails for breaches, live over a
// stream for small lists or as an async job for large ones (or --job), prints a
// summary, exports, and returns errBreachDetected (non-zero exit) when any
// address is breached and --fail-on-finding is set.
func runBreachesBulk(cmd *cobra.Command, path string) error {
	if err := validateOnly(cmd, "breached"); err != nil {
		return err
	}
	forceStream, _ := cmd.Flags().GetBool("stream")
	forceJob, _ := cmd.Flags().GetBool("job")
	if forceStream && forceJob {
		return friendlyFormatError(cmd, "choose either --stream or --job, not both")
	}
	fileName, emails, _, err := loadEmails(cmd, path)
	if err != nil {
		return err
	}
	client, err := newClient()
	if err != nil {
		return err
	}
	out := resolveResultOut(cmd, fileName)

	if breachesUseJob(len(emails), forceJob, forceStream) {
		if !forceJob && !jsonMode() {
			output.Dim.Printf("  %d emails exceeds %d, running as an async job (use --stream to force live).\n", len(emails), breachesStreamThreshold)
		}
		return runBreachesJob(cmd, client, fileName, emails, out)
	}
	return runBreachesStream(cmd, client, path, fileName, emails, out)
}

// runBreachesStream checks emails over one live SSE stream. If the stream is
// cut, the rows that already arrived are still saved.
func runBreachesStream(cmd *cobra.Command, client api.API, path, fileName string, emails []string, out string) error {
	total := len(emails)
	asJSON := jsonMode()
	if !asJSON {
		output.Header(fmt.Sprintf("Bulk Breaches: %d %s", total, plural(total, "email", "emails")))
	}

	var results []map[string]interface{}
	var rawResults []json.RawMessage
	done := 0
	start := time.Now()
	streamErr := client.BulkBreachesSearch(cmd.Context(), emails, fileName, func(ev api.BulkEvent) error {
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
		return handleStreamFailure(streamFailure{
			err:     streamErr,
			elapsed: time.Since(start),
			done:    done,
			total:   total,
			retry:   asyncRetryHint(cmd, path, "--bulk", "--job"),
		}, out, rawResults, func(out string) error { return exportBreaches(cmd, out, results) })
	}

	return finishBreaches(cmd, out, results, rawResultsArray(rawResults),
		total, countBreachedRows(results), sumCredits(results))
}

// finishBreaches prints the summary (or the JSON rows), writes the result file,
// and applies --fail-on-finding. Shared by the stream and job paths.
func finishBreaches(cmd *cobra.Command, out string, results []map[string]interface{}, jsonRows json.RawMessage, checked, breached, credits int) error {
	if jsonMode() {
		output.JSON(jsonRows)
		if out != "" {
			if err := exportBreaches(cmd, out, results); err != nil {
				return err
			}
		}
	} else {
		fmt.Println()
		fmt.Printf("  Checked %d · Breached %s · Clean %s · Credits %d\n",
			checked,
			output.Err.Sprintf("%d", breached),
			output.Success.Sprintf("%d", checked-breached),
			credits,
		)
		if err := exportBreaches(cmd, out, results); err != nil {
			return err
		}
	}
	if breached > 0 && failOnFinding(cmd) {
		return errBreachDetected
	}
	return nil
}

// breachCountOf returns the number of breaches a bulk row reports.
func breachCountOf(r map[string]interface{}) int {
	return intOf(countField(r,
		"breach_info.breach_count", "breach_info.count",
		"breach_count", "breaches", "count"))
}

// countBreachedRows counts rows exposed in at least one breach.
func countBreachedRows(rows []map[string]interface{}) int {
	n := 0
	for _, r := range rows {
		if breachCountOf(r) > 0 {
			n++
		}
	}
	return n
}

// breachNames joins the names of the breaches a row is exposed in.
func breachNames(r map[string]interface{}) string {
	arr := firstArr(r, "breach_info.services", "breaches", "breach_info.breaches")
	names := make([]string, 0, len(arr))
	for _, b := range arr {
		if n := field(asMap(b), "name", "title", "source"); n != "" {
			names = append(names, n)
		}
	}
	return strings.Join(names, " | ")
}

// breachExportColumns is the flat schema for bulk-breaches CSV/XLSX exports.
var breachExportColumns = []exportColumn{
	{"email", func(r map[string]interface{}) string { return field(r, "email", "query") }},
	{"breached", func(r map[string]interface{}) string { return yesNo(strconv.FormatBool(breachCountOf(r) > 0)) }},
	{"breach_count", func(r map[string]interface{}) string { return strconv.Itoa(breachCountOf(r)) }},
	{"breaches", breachNames},
}

// exportBreaches writes bulk-breaches results to a file, honoring --format or
// the --out extension. JSON emits the raw, nested objects.
func exportBreaches(cmd *cobra.Command, out string, results []map[string]interface{}) error {
	if _, _, _, breachedOnly := resolveOnlyFilters(cmd); breachedOnly {
		filtered := make([]map[string]interface{}, 0, len(results))
		for _, r := range results {
			if breachCountOf(r) > 0 {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}
	formatFlag, _ := cmd.Flags().GetString("format")
	format, err := resolveExportFormat(formatFlag, out)
	if err != nil {
		return friendlyFormatError(cmd, err.Error())
	}
	if strings.TrimSpace(out) == "" {
		out = fmt.Sprintf("email-breaches.%s", format)
	}

	switch format {
	case "json":
		err = writeRawJSON(out, results)
	case "xlsx":
		err = writeXLSX(out, breachExportColumns, results)
	default:
		err = writeFlatCSV(out, breachExportColumns, results)
	}
	if err != nil {
		return err
	}
	if info, statErr := os.Stat(out); statErr != nil || info.Size() == 0 {
		return fmt.Errorf("failed to write results to %s", out)
	}
	abs, absErr := filepath.Abs(out)
	if absErr != nil {
		abs = out
	}
	fmt.Fprintf(os.Stderr, "  Wrote %d %s\n", len(results), plural(len(results), "row", "rows"))
	output.SavedPath(abs)
	return nil
}
