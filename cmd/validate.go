package cmd

import (
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [email]",
	Short: "Validate an email address (free)",
	Long:  "Check whether an email address is valid and deliverable. Free — no credits used.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		data, err := client.Validate(cmd.Context(), args[0])
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

		output.Header("Validate: " + args[0])
		output.KV(
			"Validity", getStr(result, "validity"),
			"Message", getStr(result, "message"),
		)
		return nil
	},
}
