package cmd

import (
	"fmt"

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
				getStr(m, "key_preview"),
				fmt.Sprintf("%d", getInt(m, "credits_used")),
			})
		}
		output.Table([]string{"ID", "Name", "Preview", "Credits"}, rows)
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
		client, err := newClient()
		if err != nil {
			return err
		}
		permanent, _ := cmd.Flags().GetBool("permanent")
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
	keysCmd.AddCommand(keysLsCmd, keysCreateCmd, keysRevokeCmd)
}
