package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
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

var integrationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connected accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading integrations...")
		resp, err := client.ListIntegrations(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
			return err
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		items := unwrapArray(resp, "integrations")
		if len(items) == 0 {
			output.Info("No connected accounts. Start one with: encrata workflows integrations providers")
			return nil
		}
		output.Header(fmt.Sprintf("Integrations: %d", len(items)))
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				firstNonEmpty(getStr(m, "label"), getStr(m, "provider_config_key")),
				getStr(m, "provider_config_key"),
				timeField(m, "created_at"),
			})
		}
		output.Table([]string{"ID", "Label", "Provider", "Created"}, rows)
		return nil
	},
}

var integrationsProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List connectable providers (host + connect URL)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading providers...")
		resp, err := client.NangoProviders(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
			return integrationsConfigError(err)
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		var m map[string]interface{}
		if !decode(resp, &m) {
			return nil
		}
		if host := getStr(m, "host"); host != "" {
			output.KV("Host", host, "Connect URL", getStr(m, "connect_url"))
		}
		items := unwrapArray(resp, "providers")
		if len(items) == 0 {
			output.Info("No providers available.")
			return nil
		}
		output.Header(fmt.Sprintf("Providers: %d", len(items)))
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			pm, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				firstNonEmpty(getStr(pm, "provider_config_key"), getStr(pm, "unique_key"), getStr(pm, "provider")),
				firstNonEmpty(getStr(pm, "display_name"), getStr(pm, "name")),
			})
		}
		output.Table([]string{"Provider key", "Name"}, rows)
		return nil
	},
}

var integrationsSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Start a Nango Connect session (returns a session token)",
	RunE: func(cmd *cobra.Command, args []string) error {
		integration, _ := cmd.Flags().GetString("integration")
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Starting session...")
		resp, err := client.NangoSession(cmd.Context(), strings.TrimSpace(integration))
		stopSpinner(spinner)
		if err != nil {
			return integrationsConfigError(err)
		}
		// The session token is a short-lived Connect token, not an account
		// secret, but keep human output minimal; use --json to capture it.
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		var m map[string]interface{}
		if !decode(resp, &m) {
			return nil
		}
		output.Header("Connect session ready")
		output.KV(
			"Expires at", getStr(m, "expires_at"),
			"Host", getStr(m, "host"),
			"Connect URL", getStr(m, "connect_url"),
		)
		output.Info("Use --json to retrieve the session token for the Connect flow.")
		return nil
	},
}

var integrationsSaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Persist a completed Nango connection",
	RunE: func(cmd *cobra.Command, args []string) error {
		connectionID, _ := cmd.Flags().GetString("connection-id")
		providerKey, _ := cmd.Flags().GetString("provider-config-key")
		label, _ := cmd.Flags().GetString("label")
		if strings.TrimSpace(connectionID) == "" || strings.TrimSpace(providerKey) == "" {
			return friendlyFormatError(cmd, "--connection-id and --provider-config-key are required")
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		resp, err := client.NangoSave(cmd.Context(), strings.TrimSpace(connectionID), strings.TrimSpace(providerKey), strings.TrimSpace(label))
		if err != nil {
			return integrationsConfigError(err)
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		var m map[string]interface{}
		if !decode(resp, &m) {
			return nil
		}
		output.SuccessMsg("Connection saved")
		output.KV(
			"ID", getStr(m, "id"),
			"Label", firstNonEmpty(getStr(m, "label"), getStr(m, "provider_config_key")),
			"Provider", getStr(m, "provider_config_key"),
		)
		return nil
	},
}

var integrationsCreateSheetCmd = &cobra.Command{
	Use:   "create-sheet [integration-id]",
	Short: "Create a Google spreadsheet via a connected account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := strings.TrimSpace(args[0])
		title, _ := cmd.Flags().GetString("title")
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Creating spreadsheet...")
		resp, err := client.CreateIntegrationSheet(cmd.Context(), id, strings.TrimSpace(title))
		stopSpinner(spinner)
		if err != nil {
			return integrationsConfigError(err)
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		var m map[string]interface{}
		if !decode(resp, &m) {
			return nil
		}
		output.SuccessMsg("Spreadsheet created")
		output.KV(
			"Spreadsheet ID", getStr(m, "spreadsheet_id"),
			"URL", getStr(m, "spreadsheet_url"),
			"Sheet", firstNonEmpty(getStr(m, "sheet_name"), "Sheet1"),
		)
		return nil
	},
}

var integrationsDisconnectCmd = &cobra.Command{
	Use:     "disconnect [id]",
	Aliases: []string{"delete", "rm"},
	Short:   "Disconnect a connected account",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := strings.TrimSpace(args[0])
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes && !confirm("Disconnect integration "+id+"?") {
			output.Info("Aborted.")
			return nil
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		resp, err := client.DeleteIntegration(cmd.Context(), id)
		if err != nil {
			return err
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		output.SuccessMsg("Disconnected integration " + id)
		return nil
	},
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
