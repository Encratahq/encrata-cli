package cmd

import (
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var webhooksCreateCmd = &cobra.Command{
	Use:   "create [url]",
	Short: "Create a webhook endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetURL := strings.TrimSpace(args[0])
		if !strings.HasPrefix(targetURL, "https://") {
			return friendlyFormatError(cmd, "URL must use HTTPS")
		}
		rawEvents, _ := cmd.Flags().GetStringSlice("events")
		if len(rawEvents) == 0 {
			return friendlyFormatError(cmd, "provide at least one event via --events (e.g. --events lookup.completed)")
		}
		events, err := validateWebhookEvents(cmd, rawEvents)
		if err != nil {
			return err
		}
		description, _ := cmd.Flags().GetString("description")

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Creating webhook...")
		data, err := client.CreateWebhook(cmd.Context(), targetURL, description, events)
		stopSpinner(spinner)
		if err != nil {
			return webhookError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var wh map[string]interface{}
		if !decode(data, &wh) {
			return nil
		}
		output.SuccessMsg("Webhook created")
		output.KV(
			"ID", getStr(wh, "id"),
			"URL", getStr(wh, "url"),
			"Events", strings.Join(events, ", "),
			"Secret", getStr(wh, "secret"),
		)
		output.Warn.Println("  ⚠ Store this secret now — it signs every delivery and is shown only once.")
		output.Dim.Println("     Set it in your receiver as ENCRATA_WEBHOOK_SECRET.")
		output.Dim.Println("     Receiver quickstart: https://docs.encrata.com/webhooks")
		return nil
	},
}

var webhooksUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a webhook endpoint",
	Long: `Update a webhook's URL, description, events, or active state.

Only the flags you pass are changed; the rest keep their current values.

  encrata webhooks update ID --events lookup.completed,credits.low
  encrata webhooks update ID --disable
  encrata webhooks update ID --url https://example.com/new`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		enable, _ := cmd.Flags().GetBool("enable")
		disable, _ := cmd.Flags().GetBool("disable")
		if enable && disable {
			return friendlyFormatError(cmd, "choose either --enable or --disable, not both")
		}

		client, err := newClient()
		if err != nil {
			return err
		}

		// The backend PUT is a full replace, so load current values and merge in
		// only the fields the user changed.
		spinner := startSpinner("Loading webhook...")
		current, err := client.GetWebhook(cmd.Context(), id)
		stopSpinner(spinner)
		if err != nil {
			return webhookError(err)
		}
		var wh map[string]interface{}
		if !decode(current, &wh) {
			return nil
		}

		targetURL := getStr(wh, "url")
		if cmd.Flags().Changed("url") {
			u, _ := cmd.Flags().GetString("url")
			u = strings.TrimSpace(u)
			if !strings.HasPrefix(u, "https://") {
				return friendlyFormatError(cmd, "URL must use HTTPS")
			}
			targetURL = u
		}

		description := getStr(wh, "description")
		if cmd.Flags().Changed("description") {
			description, _ = cmd.Flags().GetString("description")
		}

		events := webhookEventsOf(wh["events"])
		if cmd.Flags().Changed("events") {
			raw, _ := cmd.Flags().GetStringSlice("events")
			events, err = validateWebhookEvents(cmd, raw)
			if err != nil {
				return err
			}
		}

		var isActive *bool
		if enable {
			v := true
			isActive = &v
		}
		if disable {
			v := false
			isActive = &v
		}

		spinner = startSpinner("Updating webhook...")
		data, err := client.UpdateWebhook(cmd.Context(), id, targetURL, description, events, isActive)
		stopSpinner(spinner)
		if err != nil {
			return webhookError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Webhook " + id + " updated")
		return nil
	},
}

var webhooksDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a webhook endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes && !jsonMode() {
			if !confirm(fmt.Sprintf("Delete webhook %s? This stops all deliveries.", id)) {
				output.Info("Cancelled")
				return nil
			}
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Deleting webhook...")
		data, err := client.DeleteWebhook(cmd.Context(), id)
		stopSpinner(spinner)
		if err != nil {
			return webhookError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Webhook " + id + " deleted")
		return nil
	},
}
