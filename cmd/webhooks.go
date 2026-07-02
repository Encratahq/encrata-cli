package cmd

import (
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Manage webhooks",
}

var webhooksLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List webhooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.ListWebhooks()
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		webhooks := unwrapArray(data, "webhooks")
		output.Header("Webhooks")
		if len(webhooks) == 0 {
			output.Dim.Println("  No webhooks found")
			return nil
		}
		rows := make([][]string, 0, len(webhooks))
		for _, item := range webhooks {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			active := "yes"
			if !getBool(m, "is_active") {
				active = "no"
			}
			events := make([]string, 0)
			for _, e := range getArr(m, "events") {
				events = append(events, fmt.Sprintf("%v", e))
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "url"),
				strings.Join(events, ","),
				active,
			})
		}
		output.Table([]string{"ID", "URL", "Events", "Active"}, rows)
		return nil
	},
}

var webhooksCreateCmd = &cobra.Command{
	Use:   "create [url]",
	Short: "Register a webhook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		events, _ := cmd.Flags().GetStringSlice("events")
		if len(events) == 0 {
			return fmt.Errorf("at least one --events value is required")
		}
		description, _ := cmd.Flags().GetString("description")

		data, err := client.CreateWebhook(&api.WebhookRequest{
			URL:         args[0],
			Events:      events,
			Description: description,
		})
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var m map[string]interface{}
		if decode(data, &m) {
			output.SuccessMsg("Webhook created")
			output.KV("ID", getStr(m, "id"), "Secret", getStr(m, "secret"))
			output.Warn.Println("  Store the signing secret now — it will not be shown again.")
		}
		return nil
	},
}

var webhooksUpdateCmd = &cobra.Command{
	Use:   "update [id] [url]",
	Short: "Update a webhook",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		req := &api.WebhookRequest{ID: args[0], URL: args[1]}
		if cmd.Flags().Changed("events") {
			req.Events, _ = cmd.Flags().GetStringSlice("events")
		}
		if cmd.Flags().Changed("description") {
			req.Description, _ = cmd.Flags().GetString("description")
		}
		if cmd.Flags().Changed("active") {
			active, _ := cmd.Flags().GetBool("active")
			req.IsActive = &active
		}

		data, err := client.UpdateWebhook(req)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Webhook updated: " + args[0])
		return nil
	},
}

var webhooksRmCmd = &cobra.Command{
	Use:   "rm [id]",
	Short: "Delete a webhook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.DeleteWebhook(args[0])
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Webhook deleted: " + args[0])
		return nil
	},
}

var webhooksTestCmd = &cobra.Command{
	Use:   "test [id]",
	Short: "Send a test event to a webhook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.TestWebhook(args[0])
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Test event sent: " + args[0])
		return nil
	},
}

var webhooksDeliveriesCmd = &cobra.Command{
	Use:   "deliveries [id]",
	Short: "List recent webhook deliveries",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.ListWebhookDeliveries(args[0])
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		deliveries := unwrapArray(data, "deliveries")
		output.Header("Webhook Deliveries")
		if len(deliveries) == 0 {
			output.Dim.Println("  No deliveries found")
			return nil
		}
		rows := make([][]string, 0, len(deliveries))
		for _, item := range deliveries {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "event_type"),
				getStr(m, "status"),
				fmt.Sprintf("%d", getInt(m, "response_status")),
				fmt.Sprintf("%d", getInt(m, "attempts")),
				getStr(m, "created_at"),
			})
		}
		output.Table([]string{"Event", "Status", "Response", "Attempts", "Created"}, rows)
		return nil
	},
}

func init() {
	webhooksCreateCmd.Flags().StringSlice("events", nil, "Event types to subscribe to")
	webhooksCreateCmd.Flags().String("description", "", "Webhook description")

	webhooksUpdateCmd.Flags().StringSlice("events", nil, "Event types to subscribe to")
	webhooksUpdateCmd.Flags().String("description", "", "Webhook description")
	webhooksUpdateCmd.Flags().Bool("active", true, "Whether the webhook is active")

	webhooksCmd.AddCommand(webhooksLsCmd, webhooksCreateCmd, webhooksUpdateCmd, webhooksRmCmd, webhooksTestCmd, webhooksDeliveriesCmd)
}
