package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var listsCmd = &cobra.Command{
	Use:   "lists",
	Short: "Manage contact lists",
}

var listsLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all contact lists",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		listType, _ := cmd.Flags().GetString("type")
		data, err := client.ListContactLists(listType)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}

		lists := unwrapArray(data, "lists")
		output.Header("Contact Lists")
		if len(lists) == 0 {
			output.Dim.Println("  No lists found")
			return nil
		}
		rows := make([][]string, 0, len(lists))
		for _, item := range lists {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "name"),
				getStr(m, "list_type"),
				fmt.Sprintf("%d", getInt(m, "email_count")),
			})
		}
		output.Table([]string{"ID", "Name", "Type", "Count"}, rows)
		return nil
	},
}

var listsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a contact list",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		listType, _ := cmd.Flags().GetString("type")
		targets, _ := cmd.Flags().GetStringSlice("targets")

		data, err := client.CreateContactList(&api.ListCreateRequest{
			Name:    args[0],
			Type:    listType,
			Targets: targets,
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
			output.SuccessMsg("List created")
			output.KV("ID", getStr(m, "id"), "Name", getStr(m, "name"))
		}
		return nil
	},
}

var listsGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Show a contact list",
	Args:  cobra.ExactArgs(1),
	RunE:  simpleGet((*api.Client).GetContactList, "Contact List"),
}

var listsRmCmd = &cobra.Command{
	Use:   "rm [id]",
	Short: "Delete a contact list",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.DeleteContactList(args[0])
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("List deleted: " + args[0])
		return nil
	},
}

var listsEmailsCmd = &cobra.Command{
	Use:   "emails [id]",
	Short: "List emails in a contact list",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.ListContactListEmails(args[0])
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		emails := unwrapArray(data, "emails")
		output.Header("Emails")
		if len(emails) == 0 {
			output.Dim.Println("  No emails")
			return nil
		}
		for _, e := range emails {
			if m, ok := e.(map[string]interface{}); ok {
				fmt.Printf("  • %s\n", getStr(m, "email"))
			} else {
				fmt.Printf("  • %v\n", e)
			}
		}
		return nil
	},
}

var listsAddCmd = &cobra.Command{
	Use:   "add [id] [email...]",
	Short: "Add emails to a contact list",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.AddContactListEmails(args[0], args[1:])
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
			output.SuccessMsg(fmt.Sprintf("Added %d email(s)", getInt(m, "added")))
		}
		return nil
	},
}

var listsRemoveCmd = &cobra.Command{
	Use:   "remove [id] [email...]",
	Short: "Remove emails from a contact list",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.DeleteContactListEmails(args[0], args[1:])
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
			output.SuccessMsg(fmt.Sprintf("Removed %d email(s)", getInt(m, "deleted")))
		}
		return nil
	},
}

func init() {
	listsLsCmd.Flags().String("type", "", "Filter by type (email, phone, domain, ip, company, darkweb)")
	listsCreateCmd.Flags().String("type", "", "List type (default email)")
	listsCreateCmd.Flags().StringSlice("targets", nil, "Initial targets to add")

	listsCmd.AddCommand(listsLsCmd, listsCreateCmd, listsGetCmd, listsRmCmd, listsEmailsCmd, listsAddCmd, listsRemoveCmd)
}
