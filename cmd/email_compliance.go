package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var emailComplianceCmd = &cobra.Command{
	Use:   "compliance [email]",
	Short: "Check whether you may cold-email an address, and under whose law",
	Long: `Resolve the jurisdiction that governs cold-emailing an address, its rule,
and how much to trust the attribution.

The verdict can arrive in two phases: some signals resolve within the request and
one may finish afterwards. The printed message reflects whether the answer is
provisional, confident, or unknown.

Examples:
  encrata email compliance jane@acme.com
  encrata email compliance jane@acme.com --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return emailLookup(cmd, args[0], "Compliance", "Checking compliance...",
			api.API.EmailCompliance,
			renderCompliance,
			complianceFooter)
	},
}

func init() {
	emailComplianceCmd.Flags().String("out", "", "Write the full JSON result to a file")
}

func renderCompliance(r map[string]interface{}) {
	jurisdiction := field(r, "result.country")
	if code := field(r, "result.country_code"); code != "" {
		if jurisdiction != "" {
			jurisdiction = fmt.Sprintf("%s (%s)", jurisdiction, code)
		} else {
			jurisdiction = code
		}
	}

	printNonEmptyKV(
		"Recommendation", field(r, "result.recommendation"),
		"Jurisdiction", jurisdiction,
		"Confidence", field(r, "result.confidence"),
		"Email type", field(r, "result.email_type"),
	)
	fmt.Println()

	printNonEmptyKV(
		"Main laws", listField(r, "result.main_laws"),
		"Restrictions", listField(r, "result.main_restrictions"),
	)
	if pending := listField(r, "result.pending"); pending != "" {
		output.Dim.Printf("  Still resolving: %s\n", pending)
	}

	// The server writes the message for the end user; render it as-is.
	if msg := field(r, "message"); msg != "" {
		fmt.Println()
		output.Dim.Println("  " + msg)
	}
}

// complianceFooter reports credits from the enveloped result body.
func complianceFooter(r map[string]interface{}) {
	output.Dim.Printf("  Credits used: %s\n", firstNonEmpty(field(r, "result.credits", "credits"), "0"))
}
