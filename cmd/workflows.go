package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var workflowsCmd = &cobra.Command{
	Use:   "workflows",
	Short: "Manage automation workflows",
}

var workflowsLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List workflows",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		status, _ := cmd.Flags().GetString("status")
		data, err := client.ListWorkflows(cmd.Context(), page, limit, status)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		workflows := unwrapArray(data, "workflows")
		output.Header("Workflows")
		if len(workflows) == 0 {
			output.Dim.Println("  No workflows found")
			return nil
		}
		rows := make([][]string, 0, len(workflows))
		for _, item := range workflows {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "name"),
				getStr(m, "status"),
				fmt.Sprintf("%d", getInt(m, "version")),
			})
		}
		output.Table([]string{"ID", "Name", "Status", "Version"}, rows)
		return nil
	},
}

var workflowsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a workflow",
	Long:  "Create a workflow from a template (--template-id) or from a JSON definition (--file).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		req := &api.WorkflowCreateRequest{Name: args[0]}
		req.Description, _ = cmd.Flags().GetString("description")
		req.TemplateID, _ = cmd.Flags().GetString("template-id")

		if file, _ := cmd.Flags().GetString("file"); file != "" {
			raw, err := readFileBytes(file)
			if err != nil {
				return err
			}
			var def struct {
				Trigger map[string]interface{}   `json:"trigger"`
				Steps   []map[string]interface{} `json:"steps"`
			}
			if err := json.Unmarshal(raw, &def); err != nil {
				return fmt.Errorf("invalid workflow definition: %w", err)
			}
			req.Trigger = def.Trigger
			req.Steps = def.Steps
		}

		data, err := client.CreateWorkflow(cmd.Context(), req)
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
			output.SuccessMsg("Workflow created")
			output.KV("ID", getStr(m, "id"), "Name", getStr(m, "name"))
		}
		return nil
	},
}

var workflowsGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Show a workflow",
	Args:  cobra.ExactArgs(1),
	RunE:  simpleGet((*api.Client).GetWorkflow, "Workflow"),
}

var workflowsUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a workflow",
	Long:  "Update a workflow's name, description, status, or trigger/steps (via --file). Creates a new version.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		req := &api.WorkflowUpdateRequest{}
		if cmd.Flags().Changed("name") {
			req.Name, _ = cmd.Flags().GetString("name")
		}
		if cmd.Flags().Changed("description") {
			req.Description, _ = cmd.Flags().GetString("description")
		}
		if cmd.Flags().Changed("status") {
			req.Status, _ = cmd.Flags().GetString("status")
		}
		if file, _ := cmd.Flags().GetString("file"); file != "" {
			raw, err := readFileBytes(file)
			if err != nil {
				return err
			}
			var def struct {
				Trigger map[string]interface{}   `json:"trigger"`
				Steps   []map[string]interface{} `json:"steps"`
			}
			if err := json.Unmarshal(raw, &def); err != nil {
				return fmt.Errorf("invalid workflow definition: %w", err)
			}
			req.Trigger = def.Trigger
			req.Steps = def.Steps
		}

		data, err := client.UpdateWorkflow(cmd.Context(), args[0], req)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Workflow updated: " + args[0])
		return nil
	},
}

var workflowsRunsCmd = &cobra.Command{
	Use:   "runs",
	Short: "List workflow runs",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		workflowID, _ := cmd.Flags().GetString("workflow-id")
		data, err := client.ListWorkflowRuns(cmd.Context(), page, limit, workflowID)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		runs := unwrapArray(data, "runs")
		output.Header("Workflow Runs")
		if len(runs) == 0 {
			output.Dim.Println("  No runs found")
			return nil
		}
		rows := make([][]string, 0, len(runs))
		for _, item := range runs {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "workflow_name"),
				getStr(m, "status"),
				fmt.Sprintf("%d", getInt(m, "credits_used")),
			})
		}
		output.Table([]string{"Run ID", "Workflow", "Status", "Credits"}, rows)
		return nil
	},
}

var workflowsRunCmd = &cobra.Command{
	Use:   "run [run-id]",
	Short: "Show a workflow run",
	Args:  cobra.ExactArgs(1),
	RunE:  simpleGet((*api.Client).GetWorkflowRun, "Workflow Run"),
}

var workflowsTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List workflow templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		category, _ := cmd.Flags().GetString("category")
		data, err := client.ListWorkflowTemplates(cmd.Context(), category)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		templates := unwrapArray(data, "templates")
		output.Header("Workflow Templates")
		if len(templates) == 0 {
			output.Dim.Println("  No templates found")
			return nil
		}
		rows := make([][]string, 0, len(templates))
		for _, item := range templates {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "name"),
				getStr(m, "category"),
			})
		}
		output.Table([]string{"ID", "Name", "Category"}, rows)
		return nil
	},
}

var workflowsSecretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage workflow secrets",
}

var workflowsSecretsLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List workflow secret names",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.ListWorkflowSecrets(cmd.Context())
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		secrets := unwrapArray(data, "secrets")
		output.Header("Workflow Secrets")
		if len(secrets) == 0 {
			output.Dim.Println("  No secrets found")
			return nil
		}
		for _, item := range secrets {
			if m, ok := item.(map[string]interface{}); ok {
				fmt.Printf("  • %s\n", getStr(m, "name"))
			}
		}
		return nil
	},
}

var workflowsSecretsSetCmd = &cobra.Command{
	Use:   "set [name] [value]",
	Short: "Create a workflow secret",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.CreateWorkflowSecret(cmd.Context(), args[0], args[1])
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Secret saved: " + args[0])
		return nil
	},
}

var workflowsSecretsRmCmd = &cobra.Command{
	Use:   "rm [name]",
	Short: "Delete a workflow secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.DeleteWorkflowSecret(cmd.Context(), args[0])
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Secret deleted: " + args[0])
		return nil
	},
}

func init() {
	workflowsLsCmd.Flags().Int("page", 1, "Page number")
	workflowsLsCmd.Flags().Int("limit", 20, "Items per page")
	workflowsLsCmd.Flags().String("status", "", "Filter by status (active, paused, draft)")

	workflowsCreateCmd.Flags().String("description", "", "Workflow description")
	workflowsCreateCmd.Flags().String("template-id", "", "Clone from a template ID")
	workflowsCreateCmd.Flags().StringP("file", "f", "", "JSON file with trigger and steps")

	workflowsUpdateCmd.Flags().String("name", "", "New workflow name")
	workflowsUpdateCmd.Flags().String("description", "", "New workflow description")
	workflowsUpdateCmd.Flags().String("status", "", "New status (active, paused, draft)")
	workflowsUpdateCmd.Flags().StringP("file", "f", "", "JSON file with trigger and steps")

	workflowsRunsCmd.Flags().Int("page", 1, "Page number")
	workflowsRunsCmd.Flags().Int("limit", 20, "Items per page")
	workflowsRunsCmd.Flags().String("workflow-id", "", "Filter by workflow ID")

	workflowsTemplatesCmd.Flags().String("category", "", "Filter by category")

	workflowsSecretsCmd.AddCommand(workflowsSecretsLsCmd, workflowsSecretsSetCmd, workflowsSecretsRmCmd)
	workflowsCmd.AddCommand(workflowsLsCmd, workflowsCreateCmd, workflowsUpdateCmd, workflowsGetCmd, workflowsRunsCmd, workflowsRunCmd, workflowsTemplatesCmd, workflowsSecretsCmd)
}
