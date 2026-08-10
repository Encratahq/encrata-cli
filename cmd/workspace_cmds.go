package cmd

import (
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your workspaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading workspaces...")
		data, err := client.ListWorkspaces(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
			return workspaceError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		items := unwrapArray(data, "workspaces")
		if len(items) == 0 {
			output.Info("No workspaces found.")
			return nil
		}
		output.Header(fmt.Sprintf("Workspaces: %d", len(items)))
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "name"),
				getStr(m, "slug"),
				timeField(m, "created_at"),
			})
		}
		output.Table([]string{"ID", "Name", "Slug", "Created"}, rows)
		return nil
	},
}

var workspaceCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		if name == "" {
			return friendlyFormatError(cmd, "workspace name is required")
		}
		slug, _ := cmd.Flags().GetString("slug")
		logoURL := logoURLFlag(cmd)

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Creating workspace...")
		data, err := client.CreateWorkspace(cmd.Context(), name, strings.TrimSpace(slug), logoURL)
		stopSpinner(spinner)
		if err != nil {
			return workspaceError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var ws map[string]interface{}
		if !decode(data, &ws) {
			return nil
		}
		output.SuccessMsg(fmt.Sprintf("Workspace %q created  (id: %s, slug: %s)",
			getStr(ws, "name"), getStr(ws, "id"), getStr(ws, "slug")))
		return nil
	},
}

var workspaceSwitchCmd = &cobra.Command{
	Use:   "switch [id]",
	Short: "Set your active workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Switching workspace...")
		data, err := client.SwitchWorkspace(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			return workspaceError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Switched to workspace " + args[0])
		return nil
	},
}
