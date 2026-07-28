package cmd

import (
	"github.com/Encratahq/cli/internal/api"
	"github.com/spf13/cobra"
)

var emailEnrichCmd = &cobra.Command{
	Use:   "enrich [email]",
	Short: "Validate an email and enrich it with person/company data",
	Long:  "Return validity plus enrichment data (name, company, social profiles, domain trust) for an email address.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Enrichment renders the same full detail sections as `validity --full`.
		return emailLookup(cmd, args[0], "Enrichment", "Enriching email...",
			(*api.Client).EmailEnrich,
			renderValidity(true),
			freeFooter)
	},
}
