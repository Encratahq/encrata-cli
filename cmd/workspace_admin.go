package cmd

import (
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var workspaceUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the current workspace",
	Long: `Update your current workspace's name, slug, or logo.

Acts on your CURRENT workspace unless --id is given. Admin only.
--name is required. If you omit --slug, the server regenerates the slug from
the name — pass --slug to keep a stable slug.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("name") {
			return friendlyFormatError(cmd, "--name is required")
		}
		name, _ := cmd.Flags().GetString("name")
		name = strings.TrimSpace(name)
		if name == "" {
			return friendlyFormatError(cmd, "--name cannot be empty")
		}
		slug, _ := cmd.Flags().GetString("slug")
		id, _ := cmd.Flags().GetString("id")
		logoURL := logoURLFlag(cmd)

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Updating workspace...")
		data, err := client.UpdateWorkspace(cmd.Context(), strings.TrimSpace(id), name, strings.TrimSpace(slug), logoURL)
		stopSpinner(spinner)
		if err != nil {
			return workspaceError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Workspace updated")
		return nil
	},
}

var workspaceDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete your current workspace",
	Long: `Permanently delete your CURRENT (active) workspace and all of its data.

This is irreversible and always targets the active workspace, so run
"workspace switch <id>" first to be sure you're on the right one. Admin only.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes && !jsonMode() {
			output.Warn.Println("  ⚠ This permanently deletes your CURRENT active workspace and all of its data.")
			if !confirm("Delete the current workspace?") {
				output.Info("Cancelled")
				return nil
			}
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Deleting workspace...")
		data, err := client.DeleteWorkspace(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
			return workspaceError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Workspace deleted")
		return nil
	},
}
