package cmd

import (
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var breachesCmd = &cobra.Command{
	Use:   "breaches [email]",
	Short: "Check breach exposure for an email (free)",
	Long:  "Check whether an email address appears in known data breaches. Free — no credits used.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		spinner := startSpinner("Checking breaches...")
		data, err := client.Breaches(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}

		if jsonMode() {
			output.JSON(data)
			return nil
		}

		var result map[string]interface{}
		if !decode(data, &result) {
			return nil
		}

		output.Header("Breaches: " + args[0])

		count := getInt(result, "count")
		if count == 0 {
			output.Success.Println("  ✓ No breaches found")
			return nil
		}

		output.Warn.Printf("  ⚠ Found in %d breach(es)\n", count)
		fmt.Println()

		if services := getArr(result, "services"); len(services) > 0 {
			output.Bold.Println("  Services:")
			for _, s := range services {
				fmt.Printf("    • %v\n", s)
			}
			fmt.Println()
		}

		if exposed := getArr(result, "exposed_data"); len(exposed) > 0 {
			items := make([]string, 0, len(exposed))
			for _, e := range exposed {
				items = append(items, fmt.Sprintf("%v", e))
			}
			output.KV("Exposed data", strings.Join(items, ", "))
		}
		return nil
	},
}
