package cmd

import (
	"github.com/Encratahq/cli/internal/api"
	"github.com/spf13/cobra"
)

var emailValidityCmd = &cobra.Command{
	Use:   "validity [email|file]",
	Short: "Validate an email with a live mail-server check: deliverable/undeliverable/unknown (1 credit)",
	Long: `Validate a single email address, or a whole list with --bulk.

The address is checked live against its mail server. The report shows the
status (deliverable, undeliverable or unknown), the verdict bulk jobs count
(valid, invalid, catch-all or risky), the reason, catch-all, mailbox-full,
disposable and role flags, and a score. --full adds the DNS, connection, SMTP,
TLS, catch-all test, blocklist and score details.

Charges 1 credit only when the mail server answered; repeats within the billing
window are free. An address whose server refused our check is shown as
unverified, never as invalid.

Examples:
  encrata email validity jane@acme.com
  encrata email validity jane@acme.com --full
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
			validityFooter)
	},
}

func init() {
	emailValidityCmd.Flags().Bool("full", false, "Show all validity sections")
	emailValidityCmd.Flags().Bool("bulk", false, "Validate a file (or - for STDIN) of emails")
	registerBulkFlags(emailValidityCmd, false)
}

// renderValidity returns a renderer for the validity response. Compact by
// default; full mode adds every detail section. A smtpprobe report renders with
// its own view (validity_report.go); an older result keeps the legacy one.
func renderValidity(full bool) func(map[string]interface{}) {
	return func(r map[string]interface{}) {
		if isValidityReport(r) {
			renderValidityReport(r, full)
			return
		}
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
