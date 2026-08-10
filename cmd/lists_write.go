package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

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
