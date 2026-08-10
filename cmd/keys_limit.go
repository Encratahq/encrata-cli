package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

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
