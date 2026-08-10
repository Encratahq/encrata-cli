package cmd

import (
	"github.com/Encratahq/cli/internal/api"
	"github.com/spf13/cobra"
)

var emailEnrichCmd = &cobra.Command{
	Use:   "enrich [email]",
	Short: "Validity + company & domain enrichment (the validity --full data)",
	Long:  "Return validity plus company, domain-trust and person signals for an email. For a person profile (work history, education, socials) use `email identity`.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Enrichment renders the same full detail sections as `validity --full`.
		return emailLookup(cmd, args[0], "Enrichment", "Enriching email...",
			api.API.EmailEnrich,
			renderValidity(true),
			freeFooter)
	},
}
