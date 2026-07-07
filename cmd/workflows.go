package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var workflowsCmd = &cobra.Command{
	Use:   "workflows",
	Short: "Manage automation workflows",
}

var validWorkflowStatuses = []string{"draft", "active", "paused", "archived"}

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
		spinner := startSpinner("Loading workflows...")
		data, err := client.ListWorkflows(cmd.Context(), page, limit, status)
		stopSpinner(spinner)
		if err != nil {
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

		spinner := startSpinner("Creating workflow...")
		data, err := client.CreateWorkflow(cmd.Context(), req)
		stopSpinner(spinner)
		if err != nil {
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
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading workflow...")
		data, err := client.GetWorkflow(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var workflow map[string]interface{}
		if !decode(data, &workflow) {
			return nil
		}
		output.Header("Workflow: " + args[0])
		output.KV(
			"Name", getStr(workflow, "name"),
			"Status", getStr(workflow, "status"),
			"Template", getStr(workflow, "template_id"),
			"Run Count", fmt.Sprintf("%d", getInt(workflow, "run_count")),
			"Last Run", getStr(workflow, "last_run_at"),
			"Created", getStr(workflow, "created_at"),
			"Updated", getStr(workflow, "updated_at"),
		)
		if trigger, ok := workflow["trigger"].(map[string]interface{}); ok {
			fmt.Println()
			output.Bold.Println("  Trigger:")
			output.KV("Type", getStr(trigger, "type"))
		}
		if steps, ok := workflow["steps"].([]interface{}); ok {
			fmt.Println()
			output.KV("Steps", fmt.Sprintf("%d", len(steps)))
		}
		return nil
	},
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
		req := &api.WorkflowUpdateRequest{
			Trigger: map[string]interface{}{"type": "manual", "config": map[string]interface{}{}},
			Steps:   []map[string]interface{}{},
		}
		currentData, currentErr := client.GetWorkflow(cmd.Context(), args[0])
		if currentErr == nil {
			var current map[string]interface{}
			if decode(currentData, &current) {
				req.Name = getStr(current, "name")
				req.Description = getStr(current, "description")
				req.Status = getStr(current, "status")
				if trigger, ok := current["trigger"].(map[string]interface{}); ok && trigger != nil {
					req.Trigger = trigger
				}
				if steps, ok := current["steps"].([]interface{}); ok {
					req.Steps = workflowStepsFromInterfaces(steps)
				}
			}
		}
		if cmd.Flags().Changed("name") {
			req.Name, _ = cmd.Flags().GetString("name")
		}
		if cmd.Flags().Changed("description") {
			req.Description, _ = cmd.Flags().GetString("description")
		}
		if cmd.Flags().Changed("status") {
			req.Status, _ = cmd.Flags().GetString("status")
			if err := validateWorkflowStatus(req.Status); err != nil {
				return err
			}
		}
		if req.Status == "" {
			req.Status = "draft"
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

		spinner := startSpinner("Updating workflow...")
		data, err := client.UpdateWorkflow(cmd.Context(), args[0], req)
		stopSpinner(spinner)
		if err != nil {
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
		spinner := startSpinner("Loading workflow runs...")
		data, err := client.ListWorkflowRuns(cmd.Context(), page, limit, workflowID)
		stopSpinner(spinner)
		if err != nil {
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

var workflowsTestCmd = &cobra.Command{
	Use:   "test [workflow-id]",
	Short: "Trigger a manual test run for a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Triggering workflow...")
		data, err := client.TriggerWorkflow(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var run map[string]interface{}
		if !decode(data, &run) {
			return nil
		}
		output.SuccessMsg("Workflow test run queued")
		output.KV(
			"Run ID", getStr(run, "id"),
			"Workflow ID", getStr(run, "workflow_id"),
			"Status", getStr(run, "status"),
		)
		return nil
	},
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
		spinner := startSpinner("Loading workflow templates...")
		data, err := client.ListWorkflowTemplates(cmd.Context(), category)
		stopSpinner(spinner)
		if err != nil {
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
		spinner := startSpinner("Loading workflow secrets...")
		data, err := client.ListWorkflowSecrets(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
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
		rows := make([][]string, 0, len(secrets))
		for _, item := range secrets {
			if m, ok := item.(map[string]interface{}); ok {
				rows = append(rows, []string{
					getStr(m, "id"),
					getStr(m, "name"),
					getStr(m, "secret_key"),
					getStr(m, "updated_at"),
				})
			}
		}
		output.Table([]string{"ID", "Name", "Key", "Updated"}, rows)
		return nil
		for _, item := range secrets {
			if m, ok := item.(map[string]interface{}); ok {
				fmt.Printf("  • %s\n", getStr(m, "name"))
			}
		}
		return nil
	},
}

var workflowsSecretsSetCmd = &cobra.Command{
	Use:   "set [secret-key] [value]",
	Short: "Create a workflow secret",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			name = args[0]
		}
		spinner := startSpinner("Saving workflow secret...")
		data, err := client.CreateWorkflowSecret(cmd.Context(), args[0], name, args[1])
		stopSpinner(spinner)
		if err != nil {
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var secret map[string]interface{}
		if decode(data, &secret) {
			output.SuccessMsg("Secret saved")
			output.KV("ID", getStr(secret, "id"), "Name", getStr(secret, "name"), "Key", getStr(secret, "secret_key"))
		}
		return nil
	},
}

var workflowsSecretsRmCmd = &cobra.Command{
	Use:   "rm [secret-id]",
	Short: "Delete a workflow secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Deleting workflow secret...")
		data, err := client.DeleteWorkflowSecret(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
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

	workflowsSecretsSetCmd.Flags().String("name", "", "Display name for the secret (default: secret key)")

	workflowsSecretsCmd.AddCommand(workflowsSecretsLsCmd, workflowsSecretsSetCmd, workflowsSecretsRmCmd)
	workflowsCmd.AddCommand(workflowsLsCmd, workflowsCreateCmd, workflowsUpdateCmd, workflowsGetCmd, workflowsRunsCmd, workflowsRunCmd, workflowsTestCmd, workflowsTemplatesCmd, workflowsSecretsCmd)
}

func validateWorkflowStatus(status string) error {
	for _, valid := range validWorkflowStatuses {
		if status == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid workflow status %q. Valid statuses: %s", status, strings.Join(validWorkflowStatuses, ", "))
}

func workflowStepsFromInterfaces(values []interface{}) []map[string]interface{} {
	steps := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		if step, ok := value.(map[string]interface{}); ok {
			steps = append(steps, step)
		}
	}
	return steps
}
