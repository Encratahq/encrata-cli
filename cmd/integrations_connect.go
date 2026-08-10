package cmd

import (
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

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
