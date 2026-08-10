package cmd

import (
	"strconv"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage API keys",
}

// runKeyStatus toggles an API key between enabled and disabled.
func runKeyStatus(cmd *cobra.Command, id, action string) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	verb := "Enabling"
	if action == "disable" {
		verb = "Disabling"
	}
	spinner := startSpinner(verb + " API key...")
	data, err := client.SetKeyStatus(cmd.Context(), id, action)
	stopSpinner(spinner)
	if err != nil {
		output.Error(err.Error())
		return err
	}
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	if action == "enable" {
		output.SuccessMsg("API key enabled: " + id)
	} else {
		output.SuccessMsg("API key disabled: " + id)
	}
	return nil
}

// keyStatusLabel derives a human-readable status from the key's active flags.
func keyStatusLabel(m map[string]interface{}) string {
	if b, ok := m["disabled_by_admin"].(bool); ok && b {
		return "admin-disabled"
	}
	if b, ok := m["is_active"].(bool); ok && !b {
		return "disabled"
	}
	return "enabled"
}

// keyLimitLabel renders the credit limit, or "—" when unlimited (null).
func keyLimitLabel(m map[string]interface{}) string {
	if v, ok := m["credit_limit"]; ok && v != nil {
		if f, ok := v.(float64); ok {
			return strconv.Itoa(int(f))
		}
	}
	return "—"
}

func init() {
	keysRevokeCmd.Flags().Bool("permanent", false, "Permanently delete the key instead of disabling it")
	keysRevokeCmd.Flags().Bool("yes", false, "Skip the confirmation prompt")
	keysLimitCmd.Flags().Int("credits", 0, "Credit limit for the key (must be >= 0)")
	keysLimitCmd.Flags().Bool("unlimited", false, "Remove the credit limit (unlimited usage)")
	keysCmd.AddCommand(
		keysLsCmd,
		keysCreateCmd,
		keysRenameCmd,
		keysEnableCmd,
		keysDisableCmd,
		keysRevokeCmd,
		keysLimitCmd,
	)
}
