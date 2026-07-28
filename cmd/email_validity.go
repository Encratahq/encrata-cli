package cmd

import (
	"github.com/Encratahq/cli/internal/api"
	"github.com/spf13/cobra"
)

var emailValidityCmd = &cobra.Command{
	Use:   "validity [email]",
	Short: "Check whether an email is valid and deliverable",
	Long:  "Validate a single email address. Charges 1 credit on success; invalid or failed checks are not charged.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		full, _ := cmd.Flags().GetBool("full")
		return emailLookup(cmd, args[0], "Validity", "Checking validity...",
			(*api.Client).EmailValidity,
			renderValidity(full),
			nil)
	},
}

func init() {
	emailValidityCmd.Flags().Bool("full", false, "Show all validity sections")
}

// renderValidity returns a renderer for the validity response. Compact by
// default; full mode adds every detail section.
func renderValidity(full bool) func(map[string]interface{}) {
	return func(r map[string]interface{}) {
		status := field(r, "validity", "status")
		printNonEmptyKV(
			"Email", field(r, "email"),
			"Validity", statusColor(status),
			"Deliverable", deliverableLabel(status),
			"Disposable", boolField(r, "disposable", "signals.disposable"),
			"Role", boolField(r, "role", "role_account", "signals.role"),
			"Reason", field(r, "reason", "message"),
		)
		if full {
			renderFullSections(r)
		}
	}
}
