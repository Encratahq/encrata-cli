package cmd

import (
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

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
