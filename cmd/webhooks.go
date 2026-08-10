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
