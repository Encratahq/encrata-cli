package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/config"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login [api-key]",
	Short: "Save your API key and verify it works",
	Long: `Store your Encrata API key and confirm it authenticates.

Pass the key as an argument or enter it when prompted. Create a key at
https://encrata.com/api-keys. For automation, the ENCRATA_API_KEY env var
works without logging in.

Examples:
  encrata login enc_live_xxxxxxxx
  encrata login`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var key string
		if len(args) == 1 {
			key = strings.TrimSpace(args[0])
		} else {
			fmt.Fprint(cmd.OutOrStdout(), "  Enter API key: ")
			input, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read API key: %w", err)
			}
			key = strings.TrimSpace(input)
		}
		if key == "" {
			return friendlyFormatError(cmd, "API key cannot be empty")
		}

		client := api.New(app.cfg.BaseURL, key)
		spinner := startSpinner("Verifying key...")
		data, err := client.Me(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
			output.Error("Login failed: " + err.Error())
			return err
		}

		app.cfg.APIKey = key
		if err := config.Save(app.cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		var m map[string]interface{}
		if decode(data, &m) {
			output.SuccessMsg("Logged in as " + firstNonEmpty(getStr(m, "email"), "your account"))
		} else {
			output.SuccessMsg("Logged in")
		}
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear the saved API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		app.cfg.APIKey = ""
		if err := config.Save(app.cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		output.SuccessMsg("Logged out — saved API key cleared")
		output.Dim.Println("  The ENCRATA_API_KEY env var (if set) still applies.")
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the authenticated account and active workspace",
	Long: `Show who you're authenticated as, your plan, remaining credits, role,
and the active workspace.

Example:
  encrata whoami`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading account...")
		data, err := client.Me(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var m map[string]interface{}
		if !decode(data, &m) {
			return nil
		}
		workspace := ""
		if w, ok := m["workspace"].(map[string]interface{}); ok {
			workspace = getStr(w, "name")
		}
		output.Header("Account")
		output.KV(
			"Email", getStr(m, "email"),
			"Name", getStr(m, "name"),
			"Plan", getStr(m, "plan"),
			"Credits", fmt.Sprintf("%d", getInt(m, "credits_remaining")),
			"Role", getStr(m, "role"),
			"Workspace", workspace,
		)
		return nil
	},
}
