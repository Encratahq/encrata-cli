package cmd

import (
	"fmt"
	"strconv"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage API keys",
}

var keysLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List API keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading API keys...")
		data, err := client.ListKeys(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		keys := unwrapArray(data, "keys")
		output.Header("API Keys")
		if len(keys) == 0 {
			output.Dim.Println("  No keys found")
			return nil
		}
		rows := make([][]string, 0, len(keys))
		for _, item := range keys {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "name"),
				getStr(m, "key_prefix"),
				keyStatusLabel(m),
				fmt.Sprintf("%d", getInt(m, "credits_used")),
				keyLimitLabel(m),
			})
		}
		output.Table([]string{"ID", "Name", "Prefix", "Status", "Credits", "Limit"}, rows)
		return nil
	},
}

var keysCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create an API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Creating API key...")
		data, err := client.CreateKey(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var m map[string]interface{}
		if decode(data, &m) {
			output.SuccessMsg("API key created")
			output.KV("ID", getStr(m, "id"), "Name", getStr(m, "name"), "Key", getStr(m, "key"))
			output.Warn.Println("  Store this key now — it will not be shown again.")
		}
		return nil
	},
}

var keysRevokeCmd = &cobra.Command{
	Use:   "revoke [id]",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		permanent, _ := cmd.Flags().GetBool("permanent")
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes && !jsonMode() {
			action := "revoke (disable)"
			if permanent {
				action = "permanently DELETE"
			}
			if !confirm(fmt.Sprintf("%s API key %s?", action, args[0])) {
				output.Info("Cancelled")
				return nil
			}
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Revoking API key...")
		data, err := client.RevokeKey(cmd.Context(), args[0], permanent)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		if permanent {
			output.SuccessMsg("API key permanently deleted: " + args[0])
		} else {
			output.SuccessMsg("API key revoked: " + args[0])
		}
		return nil
	},
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

var keysRenameCmd = &cobra.Command{
	Use:   "rename [id] [new-name]",
	Short: "Rename an API key",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Renaming API key...")
		data, err := client.RenameKey(cmd.Context(), args[0], args[1])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("API key renamed")
		output.KV("ID", args[0], "Name", args[1])
		return nil
	},
}

var keysEnableCmd = &cobra.Command{
	Use:   "enable [id]",
	Short: "Enable a disabled API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKeyStatus(cmd, args[0], "enable")
	},
}

var keysDisableCmd = &cobra.Command{
	Use:   "disable [id]",
	Short: "Disable an API key without deleting it",
	Long: `Disable (deactivate) an API key so it stops working but is retained.

Re-enable it later with "keys enable", or delete it for good with
"keys revoke <id> --permanent".`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKeyStatus(cmd, args[0], "disable")
	},
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

var keysLimitCmd = &cobra.Command{
	Use:   "limit [id]",
	Short: "Set or clear an API key's credit limit",
	Long: `Cap how many credits an API key may spend.

  encrata keys limit <id> --credits 5000   # set a 5000-credit cap
  encrata keys limit <id> --unlimited      # remove the cap`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		unlimited, _ := cmd.Flags().GetBool("unlimited")
		creditsSet := cmd.Flags().Changed("credits")
		credits, _ := cmd.Flags().GetInt("credits")

		if unlimited && creditsSet {
			return friendlyFormatError(cmd, "choose either --credits or --unlimited, not both")
		}
		if !unlimited && !creditsSet {
			return friendlyFormatError(cmd, "provide --credits <N> or --unlimited")
		}

		var limit *int
		if !unlimited {
			if credits < 0 {
				return friendlyFormatError(cmd, "credits must be zero or greater")
			}
			limit = &credits
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Updating credit limit...")
		data, err := client.SetKeyCreditLimit(cmd.Context(), args[0], limit)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		if limit == nil {
			output.SuccessMsg("Credit limit cleared (unlimited): " + args[0])
		} else {
			output.SuccessMsg(fmt.Sprintf("Credit limit set to %d: %s", *limit, args[0]))
		}
		return nil
	},
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
