package cmd

import (
	"github.com/Encratahq/cli/internal/api"
	"github.com/spf13/cobra"
)

var emailVerifyCmd = &cobra.Command{
	Use:   "verify [email]",
	Short: "Deep SMTP mailbox check — deliverable or not (free)",
	Long:  "Perform a deep SMTP-level verification of a single email address. Returns just a deliverability verdict (valid/invalid/accept_all/disposable/unknown) and costs no credits. For the full report + metadata use `email validity`.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return emailLookup(cmd, args[0], "Verify", "Verifying email...",
			api.API.EmailVerify,
			func(r map[string]interface{}) {
				verdict := field(r, "status", "result", "verdict", "validity")
				printNonEmptyKV(
					"Email", field(r, "email"),
					"Result", statusColor(verdict),
					"Deliverable", deliverableLabel(verdict),
				)
			},
			freeFooter)
	},
}
