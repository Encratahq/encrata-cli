package cmd

import (
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

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
