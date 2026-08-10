package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

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
