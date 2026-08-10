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
