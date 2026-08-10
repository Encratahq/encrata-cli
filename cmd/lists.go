package cmd

import (
	"encoding/json"
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
