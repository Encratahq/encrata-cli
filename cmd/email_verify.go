package cmd

import (
	"github.com/Encratahq/cli/internal/api"
	"github.com/spf13/cobra"
)

var emailVerifyCmd = &cobra.Command{
	Use:   "verify [email]",
	Short: "Deep SMTP verification of an email address",
	Long:  "Perform a deep SMTP-level verification of a single email address.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return emailLookup(cmd, args[0], "Verify", "Verifying email...",
			(*api.Client).EmailVerify,
			func(r map[string]interface{}) {
				verdict := field(r, "result", "status", "verdict", "validity")
				printNonEmptyKV(
					"Email", field(r, "email"),
					"Result", statusColor(verdict),
					"Deliverable", deliverableLabel(verdict),
					"MX found", boolField(r, "mx_found", "smtp.mx_found"),
					"SMTP check", boolField(r, "smtp_check", "smtp.check"),
					"Catch-all", boolField(r, "catch_all", "smtp.catch_all"),
					"Reason", field(r, "reason", "message"),
				)
			},
			nil)
	},
}
