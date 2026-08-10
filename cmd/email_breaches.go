package cmd

import (
	"fmt"

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
			api.API.EmailBreaches,
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
