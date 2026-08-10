package cmd

import (
	"github.com/Encratahq/cli/internal/api"
	"github.com/spf13/cobra"
)

var emailValidityCmd = &cobra.Command{
	Use:   "validity [email|file]",
	Short: "Validate an email: valid/invalid/catch-all/risky + full report (1 credit)",
	Long: `Validate a single email address, or a whole list with --bulk.

Charges 1 credit on success; invalid or failed checks are not charged.

Examples:
  encrata email validity user@example.com
  encrata email validity emails.csv --bulk
  encrata email validity emails.csv --bulk --out results.csv --only valid`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if bulk, _ := cmd.Flags().GetBool("bulk"); bulk {
			return runEmailBulk(cmd, args)
		}
		if len(args) != 1 {
			return friendlyFormatError(cmd, "provide one email address, or use --bulk with a file")
		}
		full, _ := cmd.Flags().GetBool("full")
		return emailLookup(cmd, args[0], "Validity", "Checking validity...",
			api.API.EmailValidity,
			renderValidity(full),
			nil)
	},
}

func init() {
	emailValidityCmd.Flags().Bool("full", false, "Show all validity sections")
	emailValidityCmd.Flags().Bool("bulk", false, "Validate a file (or - for STDIN) of emails")
	registerBulkFlags(emailValidityCmd, false)
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
