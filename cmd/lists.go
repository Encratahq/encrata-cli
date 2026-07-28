package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/validation"
	"github.com/spf13/cobra"
)

var listsCmd = &cobra.Command{
	Use:   "lists",
	Short: "Manage contact lists",
	Long:  `Manage reusable contact lists used by monitor and enrichment workflows.`,
}

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

var listsCreateCmd = &cobra.Command{
	Use:     "create [name]",
	Aliases: []string{"create_contact_list"},
	Short:   "Create a contact list",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		emails, err := gatherEmailsForList(cmd)
		if err != nil {
			return err
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Creating contact list...")
		data, err := client.CreateContactList(cmd.Context(), args[0], emails)
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
		output.Header("Contact List Created")
		output.KV(
			"ID", getStr(list, "id"),
			"Name", getStr(list, "name"),
			"Emails", fmt.Sprintf("%d", getInt(list, "email_count")),
		)
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

var listsDeleteCmd = &cobra.Command{
	Use:     "delete [list-id]",
	Aliases: []string{"rm", "remove", "del", "delete_contact_list"},
	Short:   "Delete a contact list",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Deleting contact list...")
		data, err := client.DeleteContactList(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Contact list deleted: " + args[0])
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

var listsAddCmd = &cobra.Command{
	Use:     "add [list-id]",
	Aliases: []string{"add_emails_to_list"},
	Short:   "Add emails to a contact list",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		emails, err := gatherEmailsForList(cmd)
		if err != nil {
			return err
		}
		if len(emails) == 0 {
			return friendlyFormatError(cmd, "provide at least one email via --emails or --file")
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Adding emails...")
		data, err := client.AddEmailsToList(cmd.Context(), args[0], emails)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var res map[string]interface{}
		if decode(data, &res) {
			output.SuccessMsg(fmt.Sprintf("Added %s email(s)", firstNonEmpty(getStr(res, "added"), getStr(res, "count"))))
			return nil
		}
		return nil
	},
}

var listsRemoveCmd = &cobra.Command{
	Use:     "remove [list-id]",
	Aliases: []string{"rm-emails", "remove_emails_from_list"},
	Short:   "Remove emails from a contact list",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		emails, err := gatherEmailsForList(cmd)
		if err != nil {
			return err
		}
		if len(emails) == 0 {
			return friendlyFormatError(cmd, "provide at least one email via --emails or --file")
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Removing emails...")
		data, err := client.RemoveEmailsFromList(cmd.Context(), args[0], emails)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var res map[string]interface{}
		if decode(data, &res) {
			output.SuccessMsg(fmt.Sprintf("Removed %s email(s)", firstNonEmpty(getStr(res, "deleted"), getStr(res, "count"))))
			return nil
		}
		return nil
	},
}

func gatherEmailsForList(cmd *cobra.Command) ([]string, error) {
	emails, _ := cmd.Flags().GetStringSlice("emails")
	file, _ := cmd.Flags().GetString("file")

	seen := map[string]bool{}
	out := make([]string, 0)
	appendEmail := func(email string) error {
		email = strings.TrimSpace(email)
		if email == "" {
			return nil
		}
		if err := validation.Email(email); err != nil {
			return friendlyFormatError(cmd, err.Error())
		}
		key := strings.ToLower(email)
		if !seen[key] {
			seen[key] = true
			out = append(out, email)
		}
		return nil
	}

	for _, email := range emails {
		if err := appendEmail(email); err != nil {
			return nil, err
		}
	}

	if file != "" {
		raw, err := readFileBytes(file)
		if err != nil {
			return nil, err
		}
		for _, email := range parseEmails(raw) {
			if err := appendEmail(email); err != nil {
				return nil, err
			}
		}
	}

	return out, nil
}

func renderContactListEmails(data json.RawMessage) bool {
	var plain []string
	if json.Unmarshal(data, &plain) == nil {
		if len(plain) == 0 {
			output.Info("No emails in this list")
			return true
		}
		rows := make([][]string, 0, len(plain))
		for _, email := range plain {
			rows = append(rows, []string{email})
		}
		output.Table([]string{"Email"}, rows)
		return true
	}

	var nested []map[string]interface{}
	if json.Unmarshal(data, &nested) == nil {
		if len(nested) == 0 {
			output.Info("No emails in this list")
			return true
		}
		rows := make([][]string, 0, len(nested))
		for _, item := range nested {
			rows = append(rows, []string{firstNonEmpty(getStr(item, "email"), getStr(item, "value"))})
		}
		output.Table([]string{"Email"}, rows)
		return true
	}

	var wrapped map[string]interface{}
	if json.Unmarshal(data, &wrapped) == nil {
		raw, ok := wrapped["emails"].([]interface{})
		if !ok {
			return false
		}
		if len(raw) == 0 {
			output.Info("No emails in this list")
			return true
		}
		rows := make([][]string, 0, len(raw))
		for _, item := range raw {
			if email, ok := item.(string); ok {
				rows = append(rows, []string{email})
				continue
			}
			if m, ok := item.(map[string]interface{}); ok {
				rows = append(rows, []string{firstNonEmpty(getStr(m, "email"), getStr(m, "value"))})
			}
		}
		output.Table([]string{"Email"}, rows)
		return true
	}
	return false
}

func init() {
	listsCreateCmd.Flags().StringSlice("emails", nil, "Initial email addresses")
	listsCreateCmd.Flags().String("file", "", "Read initial emails from a file")

	listsAddCmd.Flags().StringSlice("emails", nil, "Email addresses to add")
	listsAddCmd.Flags().String("file", "", "Read emails to add from a file")

	listsRemoveCmd.Flags().StringSlice("emails", nil, "Email addresses to remove")
	listsRemoveCmd.Flags().String("file", "", "Read emails to remove from a file")

	listsCmd.AddCommand(
		listsListCmd,
		listsCreateCmd,
		listsGetCmd,
		listsDeleteCmd,
		listsEmailsCmd,
		listsAddCmd,
		listsRemoveCmd,
	)
}
