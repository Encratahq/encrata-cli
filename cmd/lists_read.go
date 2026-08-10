package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var listsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "list_contact_lists"},
	Short:   "List contact lists",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading contact lists...")
		data, err := client.ListContactLists(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}

		items := unwrapArray(data, "lists")
		if len(items) == 0 {
			output.Info("No contact lists found")
			return nil
		}

		output.Header(fmt.Sprintf("Contact Lists: %d", len(items)))
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "name"),
				fmt.Sprintf("%d", getInt(m, "email_count")),
				timeField(m, "created_at"),
			})
		}
		output.Table([]string{"ID", "Name", "Emails", "Created"}, rows)
		return nil
	},
}

var listsGetCmd = &cobra.Command{
	Use:     "get [list-id]",
	Aliases: []string{"show", "get_contact_list"},
	Short:   "Get contact list details",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading contact list...")
		data, err := client.GetContactList(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}

		var list map[string]interface{}
		if !decode(data, &list) {
			return nil
		}
		output.Header("Contact List: " + args[0])
		output.KV(
			"ID", getStr(list, "id"),
			"Name", getStr(list, "name"),
			"Emails", fmt.Sprintf("%d", getInt(list, "email_count")),
			"Created", timeField(list, "created_at"),
		)
		return nil
	},
}

var listsEmailsCmd = &cobra.Command{
	Use:     "emails [list-id]",
	Aliases: []string{"list_contact_list_emails"},
	Short:   "List emails in a contact list",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading list emails...")
		data, err := client.ListContactListEmails(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}

		output.Header("List Emails: " + args[0])
		if renderContactListEmails(data) {
			return nil
		}
		output.JSON(data)
		return nil
	},
}
