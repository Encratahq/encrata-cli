package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// validWorkspaceRoles mirrors the backend allow-list. Keep in sync with
// encrata/backend/internal/validator/workspace.go.
var validWorkspaceRoles = []string{"admin", "tech", "readonly"}

var workspaceCmd = &cobra.Command{
	Use:     "workspace",
	Aliases: []string{"ws"},
	Short:   "Manage workspaces and members",
	Long: `Manage workspaces and their members.

Member, update, and delete operations act on your CURRENT (active) workspace,
which is tracked server-side per user. Use "workspace switch <id>" to change it
before running those commands.

Examples:
  encrata workspace list
  encrata workspace create "Acme Inc"
  encrata workspace switch WORKSPACE_ID
  encrata workspace members
  encrata workspace members invite teammate@acme.com --role tech`,
}

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

// ── members ────────────────────────────────────────────────────────────────

var workspaceMembersCmd = &cobra.Command{
	Use:   "members",
	Short: "Manage members of the current workspace",
	RunE:  runListMembers,
}

var workspaceMembersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List members of the current workspace",
	RunE:  runListMembers,
}

func runListMembers(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	spinner := startSpinner("Loading members...")
	data, err := client.ListWorkspaceMembers(cmd.Context())
	stopSpinner(spinner)
	if err != nil {
		return workspaceError(err)
	}
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	items := unwrapArray(data, "members")
	if len(items) == 0 {
		output.Info("No members yet.")
		return nil
	}
	output.Header(fmt.Sprintf("Members: %d", len(items)))
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		rows = append(rows, []string{
			getStr(m, "id"),
			firstNonEmpty(getStr(m, "email"), "—"),
			getStr(m, "role"),
			getStr(m, "status"),
			timeField(m, "joined_at"),
		})
	}
	output.Table([]string{"ID", "Email", "Role", "Status", "Joined"}, rows)
	return nil
}

var workspaceMembersInviteCmd = &cobra.Command{
	Use:   "invite [email]",
	Short: "Invite a member to the current workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		roleFlag, _ := cmd.Flags().GetString("role")
		role, err := validateWorkspaceRole(cmd, roleFlag)
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Inviting member...")
		data, err := client.InviteWorkspaceMember(cmd.Context(), strings.TrimSpace(args[0]), role)
		stopSpinner(spinner)
		if err != nil {
			return workspaceError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg(fmt.Sprintf("Invited %s as %s — an invite email has been sent.", args[0], role))
		return nil
	},
}

var workspaceMembersSetRoleCmd = &cobra.Command{
	Use:   "set-role [member-id]",
	Short: "Change a member's role in the current workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		roleFlag, _ := cmd.Flags().GetString("role")
		role, err := validateWorkspaceRole(cmd, roleFlag)
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Updating member role...")
		data, err := client.SetWorkspaceMemberRole(cmd.Context(), args[0], role)
		stopSpinner(spinner)
		if err != nil {
			return workspaceError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg(fmt.Sprintf("Updated role for %s to %s", args[0], role))
		return nil
	},
}

var workspaceMembersRemoveCmd = &cobra.Command{
	Use:   "remove [member-id]",
	Short: "Remove a member from the current workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes && !jsonMode() {
			if !confirm(fmt.Sprintf("Remove member %s from this workspace?", id)) {
				output.Info("Cancelled")
				return nil
			}
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Removing member...")
		data, err := client.RemoveWorkspaceMember(cmd.Context(), id)
		stopSpinner(spinner)
		if err != nil {
			return workspaceError(err)
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Member removed")
		return nil
	},
}

// validateWorkspaceRole validates a role against the backend allow-list.
func validateWorkspaceRole(cmd *cobra.Command, role string) (string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	for _, r := range validWorkspaceRoles {
		if r == role {
			return role, nil
		}
	}
	return "", friendlyFormatError(cmd, fmt.Sprintf(
		"Invalid role %q. Valid roles: %s", role, strings.Join(validWorkspaceRoles, ", ")))
}

// logoURLFlag returns a pointer to the --logo-url value when set, else nil.
func logoURLFlag(cmd *cobra.Command) *string {
	if !cmd.Flags().Changed("logo-url") {
		return nil
	}
	v, _ := cmd.Flags().GetString("logo-url")
	return &v
}

// workspaceError prints a workspace-specific message for common failures and
// returns the original error so the command exits non-zero.
func workspaceError(err error) error {
	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.StatusCode == 401:
			output.Error("Not authenticated. Run: encrata config set-key")
		case apiErr.StatusCode == 400 && strings.Contains(strings.ToLower(apiErr.Message), "no workspace selected"):
			output.Error("No active workspace. Run: encrata workspace switch <id>")
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

func init() {
	workspaceCreateCmd.Flags().String("slug", "", "Custom slug (auto-generated when omitted)")
	workspaceCreateCmd.Flags().String("logo-url", "", "Logo URL")

	workspaceUpdateCmd.Flags().String("name", "", "New workspace name (required)")
	workspaceUpdateCmd.Flags().String("slug", "", "New slug (regenerated from name if omitted)")
	workspaceUpdateCmd.Flags().String("logo-url", "", "New logo URL")
	workspaceUpdateCmd.Flags().String("id", "", "Target workspace ID (defaults to current)")

	workspaceDeleteCmd.Flags().Bool("yes", false, "Skip the confirmation prompt")

	workspaceMembersInviteCmd.Flags().String("role", "readonly", "Role: admin, tech, or readonly")
	workspaceMembersSetRoleCmd.Flags().String("role", "", "Role: admin, tech, or readonly")
	workspaceMembersRemoveCmd.Flags().Bool("yes", false, "Skip the confirmation prompt")

	workspaceMembersCmd.AddCommand(
		workspaceMembersListCmd,
		workspaceMembersInviteCmd,
		workspaceMembersSetRoleCmd,
		workspaceMembersRemoveCmd,
	)

	workspaceCmd.AddCommand(
		workspaceListCmd,
		workspaceCreateCmd,
		workspaceSwitchCmd,
		workspaceUpdateCmd,
		workspaceDeleteCmd,
		workspaceMembersCmd,
	)
}
