package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var emailBreachesCmd = &cobra.Command{
	Use:   "breaches [email]",
	Short: "Check whether an email appears in known data breaches",
	Long:  "List known data breaches an email address is exposed in.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		full, _ := cmd.Flags().GetBool("full")
		return emailLookup(cmd, args[0], "Breaches", "Checking breaches...",
			(*api.Client).EmailBreaches,
			renderBreaches(full),
			nil)
	},
}

func init() {
	emailBreachesCmd.Flags().Bool("full", false, "Show exposed/registered services")
}

func renderBreaches(full bool) func(map[string]interface{}) {
	return func(r map[string]interface{}) {
		count := intOf(countField(r, "breach_count", "breaches", "count"))
		label := output.Success.Sprint("0")
		if count > 0 {
			label = output.Err.Sprint(fmt.Sprintf("%d", count))
		}
		output.KV("Breaches found", label)
		fmt.Println()

		renderBreachTable(r)

		if full {
			output.KV(
				"Exposed services", listField(r, "exposed_services", "services"),
				"Registered services", listField(r, "registered_services"),
			)
		}
	}
}

// renderBreachTable prints a Name | Date | Exposed data table when breaches exist.
func renderBreachTable(r map[string]interface{}) {
	arr := firstArr(r, "breaches")
	if len(arr) == 0 {
		return
	}
	rows := make([][]string, 0, len(arr))
	for _, b := range arr {
		m := asMap(b)
		rows = append(rows, []string{
			firstNonEmpty(field(m, "name", "title", "source"), "—"),
			firstNonEmpty(timeField(m, "date", "breach_date", "added_date"), "—"),
			firstNonEmpty(listField(m, "exposed_data", "data_classes", "data"), "—"),
		})
	}
	output.Table([]string{"Name", "Date", "Exposed data"}, rows)
	fmt.Println()
}
