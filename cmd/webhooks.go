package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// validWebhookEvents mirrors the backend allow-list. Keep in sync with
// encrata/backend/internal/validator/webhook.go.
var validWebhookEvents = []string{
	"lookup.completed",
	"apikey.created",
	"apikey.revoked",
	"credits.low",
	"credits.exhausted",
}

var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Manage webhook endpoints",
	Long: `Register HTTPS endpoints that receive event notifications from Encrata.

Valid events: ` + strings.Join(validWebhookEvents, ", ") + `

Examples:
  encrata webhooks create https://example.com/hook --events lookup.completed
  encrata webhooks list
  encrata webhooks test WEBHOOK_ID
  encrata webhooks deliveries WEBHOOK_ID`,
}

var webhooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhook endpoints",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading webhooks...")
		data, err := client.ListWebhooks(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
			return webhookError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		items := unwrapArray(data, "webhooks")
		if len(items) == 0 {
			output.Info("No webhooks yet. Create one with: encrata webhooks create <url> --events lookup.completed")
			return nil
		}
		output.Header(fmt.Sprintf("Webhooks: %d", len(items)))
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "url"),
				strings.Join(webhookEventsOf(m["events"]), ", "),
				webhookStatusLabel(m),
				timeField(m, "created_at"),
			})
		}
		output.Table([]string{"ID", "URL", "Events", "Status", "Created"}, rows)
		return nil
	},
}

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

var webhooksTestCmd = &cobra.Command{
	Use:   "test [id]",
	Short: "Send a test event to a webhook endpoint",
	Long: `Deliver a test event to a webhook and report the real HTTP result.

The test is sent using the webhook's first subscribed event with data.test = true.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Sending test event...")
		data, err := client.TestWebhook(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			return webhookError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var res map[string]interface{}
		if !decode(data, &res) {
			return nil
		}
		targetURL := getStr(res, "url")
		if getStr(res, "status") == "delivered" {
			output.SuccessMsg(fmt.Sprintf("Test delivered to %s (HTTP %d)", targetURL, getInt(res, "status_code")))
			return nil
		}
		if errMsg := getStr(res, "error"); errMsg != "" {
			output.Error(fmt.Sprintf("Could not reach %s: %s", targetURL, errMsg))
			return nil
		}
		output.Error(fmt.Sprintf("Test failed — %s returned HTTP %d", targetURL, getInt(res, "status_code")))
		if body := getStr(res, "response_body"); body != "" {
			if len(body) > 200 {
				body = body[:200] + "…"
			}
			output.Dim.Println("  Response: " + body)
		}
		return nil
	},
}

var webhooksDeliveriesCmd = &cobra.Command{
	Use:   "deliveries [id]",
	Short: "Show recent delivery attempts for a webhook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading deliveries...")
		data, err := client.ListWebhookDeliveries(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			return webhookError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		items := unwrapArray(data, "deliveries")
		if len(items) == 0 {
			output.Info("No deliveries yet for this webhook.")
			return nil
		}
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}
		output.Header(fmt.Sprintf("Deliveries: %d", len(items)))
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			httpCol := "—"
			if v, ok := m["response_status"]; ok && v != nil {
				httpCol = fmt.Sprintf("%d", getInt(m, "response_status"))
			}
			rows = append(rows, []string{
				getStr(m, "event_type"),
				webhookDeliveryStatus(getStr(m, "status")),
				httpCol,
				fmt.Sprintf("%d", getInt(m, "attempts")),
				timeField(m, "last_attempt_at", "created_at"),
			})
		}
		output.Table([]string{"Event", "Status", "HTTP", "Attempts", "Time"}, rows)
		return nil
	},
}

// validateWebhookEvents trims, validates against the allow-list, and de-dupes
// the requested events, returning a friendly error listing valid values.
func validateWebhookEvents(cmd *cobra.Command, events []string) ([]string, error) {
	valid := make(map[string]bool, len(validWebhookEvents))
	for _, e := range validWebhookEvents {
		valid[e] = true
	}
	seen := make(map[string]bool, len(events))
	out := make([]string, 0, len(events))
	for _, e := range events {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !valid[e] {
			return nil, friendlyFormatError(cmd, fmt.Sprintf(
				"Invalid event %q. Valid events: %s", e, strings.Join(validWebhookEvents, ", ")))
		}
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		return nil, friendlyFormatError(cmd, "provide at least one event via --events")
	}
	return out, nil
}

// webhookEventsOf converts a decoded JSON events value to a string slice.
func webhookEventsOf(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// webhookStatusLabel renders active/disabled from the is_active flag.
func webhookStatusLabel(m map[string]interface{}) string {
	if b, ok := m["is_active"].(bool); ok && b {
		return output.Success.Sprint("active")
	}
	return output.Dim.Sprint("disabled")
}

// webhookDeliveryStatus colors a delivery status label.
func webhookDeliveryStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "delivered", "success", "ok":
		return output.Success.Sprint(s)
	case "failed", "error":
		return output.Err.Sprint(s)
	case "retrying", "pending", "queued":
		return output.Warn.Sprint(s)
	default:
		return s
	}
}

// webhookError prints a webhook-specific message for common failures and
// returns the original error so the command exits non-zero.
func webhookError(err error) error {
	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.StatusCode == 401 || apiErr.StatusCode == 403:
			output.Error("You don't have permission to manage webhooks in this workspace.")
		case apiErr.StatusCode == 400 && strings.Contains(strings.ToLower(apiErr.Message), "workspace"):
			output.Error("No workspace selected. Select a workspace in the Encrata dashboard first.")
		case apiErr.StatusCode >= 500:
			output.Error(fmt.Sprintf("Encrata API error (%d): %s", apiErr.StatusCode, apiErr.Message))
		default:
			output.Error(apiErr.Message)
		}
		return err
	}
	output.Error(err.Error())
	return err
}

// confirm prompts for a y/N answer on STDIN, defaulting to no.
func confirm(prompt string) bool {
	fmt.Printf("  %s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(input))
	return answer == "y" || answer == "yes"
}

func init() {
	webhooksCreateCmd.Flags().StringSlice("events", nil, "Events to subscribe to (comma-separated)")
	webhooksCreateCmd.Flags().String("description", "", "Optional description")

	webhooksUpdateCmd.Flags().String("url", "", "New HTTPS endpoint URL")
	webhooksUpdateCmd.Flags().StringSlice("events", nil, "Replace subscribed events (comma-separated)")
	webhooksUpdateCmd.Flags().String("description", "", "New description")
	webhooksUpdateCmd.Flags().Bool("enable", false, "Enable (activate) the webhook")
	webhooksUpdateCmd.Flags().Bool("disable", false, "Disable (deactivate) the webhook")

	webhooksDeleteCmd.Flags().Bool("yes", false, "Skip the confirmation prompt")

	webhooksDeliveriesCmd.Flags().Int("limit", 20, "Maximum number of deliveries to show")

	webhooksCmd.AddCommand(
		webhooksListCmd,
		webhooksCreateCmd,
		webhooksUpdateCmd,
		webhooksDeleteCmd,
		webhooksTestCmd,
		webhooksDeliveriesCmd,
	)
}
