package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var emailBreachesCmd = &cobra.Command{
	Use:   "breaches [email|file]",
	Short: "Check whether an email appears in known data breaches",
	Long: `List known data breaches an email address is exposed in.

Pass a single email to check it, or use --bulk with a file (or - for STDIN)
to stream a whole list. Add --fail-on-finding to exit with code 2 when any
email is breached (otherwise the command exits 0).

Examples:
  encrata email breaches user@example.com
  encrata email breaches emails.csv --bulk
  encrata email breaches emails.csv --bulk --out breaches.csv`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bulk, _ := cmd.Flags().GetBool("bulk")
		if bulk {
			path := ""
			if len(args) == 1 {
				path = args[0]
			}
			return runBreachesBulk(cmd, path)
		}
		if len(args) != 1 {
			return friendlyFormatError(cmd, "provide one email address, or use --bulk with a file")
		}
		full, _ := cmd.Flags().GetBool("full")
		return emailLookup(cmd, args[0], "Breaches", "Checking breaches...",
			(*api.Client).EmailBreaches,
			renderBreaches(full),
			nil)
	},
}

func init() {
	emailBreachesCmd.Flags().Bool("full", false, "Show exposed/registered services")
	emailBreachesCmd.Flags().Bool("bulk", false, "Check a file (or - for STDIN) of emails via streaming")
	emailBreachesCmd.Flags().String("format", "", "Bulk export format: csv, xlsx, or json (default: inferred from --out)")
	emailBreachesCmd.Flags().String("only", "", "Export only rows matching: breached")
	emailBreachesCmd.Flags().Bool("fail-on-finding", false, "Exit with code 2 if any email is found in a breach (--bulk)")
}

func renderBreaches(full bool) func(map[string]interface{}) {
	return func(r map[string]interface{}) {
		count := intOf(countField(r,
			"breach_info.breach_count", "breach_info.count",
			"breach_count", "breaches", "count"))
		label := output.Success.Sprint("0")
		if count > 0 {
			label = output.Err.Sprint(fmt.Sprintf("%d", count))
		}
		output.KV("Breaches found", label)
		fmt.Println()

		renderBreachTable(r)

		if full {
			printNonEmptyKV(
				"Exposed data", listField(r, "breach_info.exposed_data", "exposed_data", "exposed_services"),
				"Registered services", listField(r, "registered_services.services", "registered_services"),
			)
		}
	}
}

// renderBreachTable prints a Name | Date | Exposed data table when breaches exist.
func renderBreachTable(r map[string]interface{}) {
	arr := firstArr(r, "breach_info.services", "breaches", "breach_info.breaches")
	if len(arr) == 0 {
		return
	}
	rows := make([][]string, 0, len(arr))
	for _, b := range arr {
		m := asMap(b)
		rows = append(rows, []string{
			firstNonEmpty(field(m, "name", "title", "source"), "—"),
			firstNonEmpty(timeField(m, "breach_date", "date", "added_date"), "—"),
			firstNonEmpty(listField(m, "data_types", "exposed_data", "data_classes", "data"), "—"),
		})
	}
	output.Table([]string{"Name", "Date", "Exposed data"}, rows)
	fmt.Println()
}

// runBreachesBulk streams breach checks for a file/STDIN list of emails, prints
// a summary table, optionally exports, and returns errBreachDetected (non-zero
// exit) when any address is found in a breach.
func runBreachesBulk(cmd *cobra.Command, path string) error {
	if err := validateOnly(cmd, "breached"); err != nil {
		return err
	}
	fileName, emails, _, err := loadEmails(cmd, path)
	if err != nil {
		return err
	}
	client, err := newClient()
	if err != nil {
		return err
	}
	out, _ := cmd.Flags().GetString("out")

	total := len(emails)
	asJSON := jsonMode()
	if !asJSON {
		output.Header(fmt.Sprintf("Bulk Breaches: %d %s", total, plural(total, "email", "emails")))
	}

	var results []map[string]interface{}
	var rawResults []json.RawMessage
	done := 0
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
		output.Error(streamErr.Error())
		return streamErr
	}

	breached := countBreachedRows(results)

	if asJSON {
		output.JSON(rawResultsArray(rawResults))
		if out != "" {
			if err := exportBreaches(cmd, out, results); err != nil {
				return err
			}
		}
		if breached > 0 && failOnFinding(cmd) {
			return errBreachDetected
		}
		return nil
	}

	fmt.Println()
	clean := total - breached
	fmt.Printf("  Checked %d · Breached %s · Clean %s\n",
		total,
		output.Err.Sprintf("%d", breached),
		output.Success.Sprintf("%d", clean),
	)

	if out != "" {
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
	if strings.TrimSpace(out) == "" && (format == "csv" || format == "xlsx") {
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
