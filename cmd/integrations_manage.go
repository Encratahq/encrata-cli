package cmd

import (
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var integrationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connected accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading integrations...")
		resp, err := client.ListIntegrations(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
			return err
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		items := unwrapArray(resp, "integrations")
		if len(items) == 0 {
			output.Info("No connected accounts. Start one with: encrata workflows integrations providers")
			return nil
		}
		output.Header(fmt.Sprintf("Integrations: %d", len(items)))
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				firstNonEmpty(getStr(m, "label"), getStr(m, "provider_config_key")),
				getStr(m, "provider_config_key"),
				timeField(m, "created_at"),
			})
		}
		output.Table([]string{"ID", "Label", "Provider", "Created"}, rows)
		return nil
	},
}

var integrationsCreateSheetCmd = &cobra.Command{
	Use:   "create-sheet [integration-id]",
	Short: "Create a Google spreadsheet via a connected account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := strings.TrimSpace(args[0])
		title, _ := cmd.Flags().GetString("title")
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Creating spreadsheet...")
		resp, err := client.CreateIntegrationSheet(cmd.Context(), id, strings.TrimSpace(title))
		stopSpinner(spinner)
		if err != nil {
			return integrationsConfigError(err)
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		var m map[string]interface{}
		if !decode(resp, &m) {
			return nil
		}
		output.SuccessMsg("Spreadsheet created")
		output.KV(
			"Spreadsheet ID", getStr(m, "spreadsheet_id"),
			"URL", getStr(m, "spreadsheet_url"),
			"Sheet", firstNonEmpty(getStr(m, "sheet_name"), "Sheet1"),
		)
		return nil
	},
}

var integrationsDisconnectCmd = &cobra.Command{
	Use:     "disconnect [id]",
	Aliases: []string{"delete", "rm"},
	Short:   "Disconnect a connected account",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := strings.TrimSpace(args[0])
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes && !confirm("Disconnect integration "+id+"?") {
			output.Info("Aborted.")
			return nil
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		resp, err := client.DeleteIntegration(cmd.Context(), id)
		if err != nil {
			return err
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		output.SuccessMsg("Disconnected integration " + id)
		return nil
	},
}
