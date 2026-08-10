package cmd

import (
	"errors"
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/spf13/cobra"
)

// integrationsCmd manages connected export destinations (Google Sheets, HubSpot,
// Salesforce, ...) via Nango. Registered as a subcommand of 'workflows'.
// Tokens/secrets are never printed.
var integrationsCmd = &cobra.Command{
	Use:     "integrations",
	Aliases: []string{"int"},
	Short:   "Manage connected export destinations (Nango)",
	Long: `List and manage connected accounts used as workflow export destinations.

Connecting a new account happens in the browser (Nango Connect). Use 'providers'
and 'session' to start that flow, then 'save' to persist the resulting
connection. Tokens and secrets are never displayed.`,
}

// integrationsConfigError turns the backend's 503 "integrations are not
// configured" into a clearer hint without leaking anything.
func integrationsConfigError(err error) error {
	var apiErr *api.Error
	if errors.As(err, &apiErr) && apiErr.StatusCode == 503 {
		return fmt.Errorf("integrations are not configured on this server (Nango is unavailable)")
	}
	return err
}

func init() {
	integrationsSessionCmd.Flags().String("integration", "", "Provider config key to connect (optional)")
	integrationsSaveCmd.Flags().String("connection-id", "", "Nango connection id (required)")
	integrationsSaveCmd.Flags().String("provider-config-key", "", "Nango provider config key (required)")
	integrationsSaveCmd.Flags().String("label", "", "Human-friendly label for the connection")
	integrationsCreateSheetCmd.Flags().String("title", "", "Spreadsheet title (default \"Encrata Enriched\")")
	integrationsDisconnectCmd.Flags().Bool("yes", false, "Skip confirmation")

	integrationsCmd.AddCommand(
		integrationsListCmd,
		integrationsProvidersCmd,
		integrationsSessionCmd,
		integrationsSaveCmd,
		integrationsCreateSheetCmd,
		integrationsDisconnectCmd,
	)
}
