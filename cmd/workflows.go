package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// workflowsCmd groups bulk file enrichment (upload → run → status → download)
// and the integrations subtree. Endpoints live under /api/workflows/* and are
// mirrored byte-for-byte from encrata/backend/internal/handlers.
var workflowsCmd = &cobra.Command{
	Use:     "workflows",
	Aliases: []string{"wf"},
	Short:   "Bulk enrichment: upload a file, run it, download results",
	Long: `Run bulk enrichment over a CSV/TXT/XLSX of emails.

Flow:
  1. encrata workflows upload emails.csv                 → file id
  2. encrata workflows run WORKFLOW_ID --file FILE_ID    → run id
  3. encrata workflows status RUN_ID                     → watch status
  4. encrata workflows download RUN_ID --out results.csv → results CSV

Connected export destinations (Google Sheets, HubSpot, ...) are managed under
'encrata workflows integrations'.`,
}

var workflowsUploadCmd = &cobra.Command{
	Use:   "upload [file]",
	Short: "Upload a CSV/TXT/XLSX of emails for bulk enrichment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := strings.TrimSpace(args[0])
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		workflowID, _ := cmd.Flags().GetString("workflow-id")

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Uploading file...")
		resp, err := client.UploadWorkflowFile(cmd.Context(), filepath.Base(path), data, workflowID)
		stopSpinner(spinner)
		if err != nil {
			return err
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		var m map[string]interface{}
		if !decode(resp, &m) {
			return nil
		}
		output.Header("File uploaded")
		output.KV(
			"ID", getStr(m, "id"),
			"Filename", getStr(m, "filename"),
			"Rows", strconv.Itoa(getInt(m, "row_count")),
			"Identifier", firstNonEmpty(getStr(m, "identifier_type"), "email"),
			"Column", firstNonEmpty(getStr(m, "identifier_column"), "email"),
		)
		output.Info("Next: encrata workflows run <workflow-id> --file " + getStr(m, "id"))
		return nil
	},
}

var workflowsRunCmd = &cobra.Command{
	Use:   "run [workflow-id]",
	Short: "Trigger a bulk run over an uploaded file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowID := strings.TrimSpace(args[0])
		fileID, _ := cmd.Flags().GetString("file")
		if strings.TrimSpace(fileID) == "" {
			return friendlyFormatError(cmd, "--file <file-id> is required (from 'encrata workflows upload')")
		}
		idem, _ := cmd.Flags().GetString("idempotency-key")

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Starting run...")
		resp, err := client.RunWorkflow(cmd.Context(), workflowID, strings.TrimSpace(fileID), strings.TrimSpace(idem))
		stopSpinner(spinner)
		if err != nil {
			return err
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		var m map[string]interface{}
		if !decode(resp, &m) {
			return nil
		}
		output.Header("Run started")
		printRunKV(m)
		output.Info("Track it with: encrata workflows status " + getStr(m, "id"))
		return nil
	},
}

var workflowsStatusCmd = &cobra.Command{
	Use:   "status [run-id]",
	Short: "Show a run's status, credits and step breakdown",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := strings.TrimSpace(args[0])
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading run...")
		resp, err := client.GetWorkflowRun(cmd.Context(), runID)
		stopSpinner(spinner)
		if err != nil {
			return err
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		var wrapper struct {
			Run   map[string]interface{}   `json:"run"`
			Steps []map[string]interface{} `json:"steps"`
		}
		if !decode(resp, &wrapper) {
			return nil
		}
		output.Header("Run status")
		printRunKV(wrapper.Run)
		if len(wrapper.Steps) > 0 {
			rows := make([][]string, 0, len(wrapper.Steps))
			for _, s := range wrapper.Steps {
				rows = append(rows, []string{
					getStr(s, "step_id"),
					getStr(s, "step_type"),
					getStr(s, "status"),
					strconv.Itoa(getInt(s, "duration_ms")) + "ms",
					getStr(s, "error"),
				})
			}
			output.Table([]string{"Step", "Type", "Status", "Duration", "Error"}, rows)
		}
		return nil
	},
}

var workflowsRunsCmd = &cobra.Command{
	Use:   "runs",
	Short: "List recent runs",
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowID, _ := cmd.Flags().GetString("workflow-id")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading runs...")
		resp, err := client.ListWorkflowRuns(cmd.Context(), workflowID, limit, offset)
		stopSpinner(spinner)
		if err != nil {
			return err
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		items := unwrapArray(resp, "runs")
		if len(items) == 0 {
			output.Info("No runs yet. Start one with: encrata workflows run <workflow-id> --file <file-id>")
			return nil
		}
		output.Header(fmt.Sprintf("Runs: %d", len(items)))
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "status"),
				strconv.Itoa(getInt(m, "output_rows")),
				strconv.Itoa(getInt(m, "credits_used")),
				timeField(m, "created_at"),
			})
		}
		output.Table([]string{"Run ID", "Status", "Rows", "Credits", "Created"}, rows)
		return nil
	},
}

var workflowsCancelCmd = &cobra.Command{
	Use:   "cancel [run-id]",
	Short: "Request cancellation of a run",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := strings.TrimSpace(args[0])
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes && !confirm("Cancel run "+runID+"?") {
			output.Info("Aborted.")
			return nil
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		resp, err := client.CancelWorkflowRun(cmd.Context(), runID)
		if err != nil {
			return err
		}
		if jsonMode() {
			output.JSON(resp)
			return nil
		}
		output.SuccessMsg("Cancellation requested for run " + runID)
		return nil
	},
}

var workflowsDownloadCmd = &cobra.Command{
	Use:   "download [run-id]",
	Short: "Download a run's enriched results CSV",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := strings.TrimSpace(args[0])
		out, _ := cmd.Flags().GetString("out")
		if strings.TrimSpace(out) == "" {
			out = "run-" + runID + ".csv"
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Downloading results...")
		data, err := client.DownloadWorkflowRunOutput(cmd.Context(), runID)
		stopSpinner(spinner)
		if err != nil {
			return err
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", out, err)
		}
		output.SuccessMsg(fmt.Sprintf("Saved %d bytes to %s", len(data), out))
		return nil
	},
}

// printRunKV renders the shared WorkflowRun fields as a key/value card.
func printRunKV(m map[string]interface{}) {
	pairs := []string{
		"Run ID", getStr(m, "id"),
		"Workflow", firstNonEmpty(getStr(m, "workflow_name"), getStr(m, "workflow_id")),
		"Status", getStr(m, "status"),
		"Trigger", getStr(m, "trigger_type"),
		"Credits", strconv.Itoa(getInt(m, "credits_used")),
		"Output rows", strconv.Itoa(getInt(m, "output_rows")),
	}
	if e := getStr(m, "error"); e != "" {
		pairs = append(pairs, "Error", e)
	}
	pairs = append(pairs, "Created", timeField(m, "created_at"))
	output.KV(pairs...)
}

func init() {
	workflowsUploadCmd.Flags().String("workflow-id", "", "Attach the file to an existing workflow")
	workflowsRunCmd.Flags().String("file", "", "Uploaded file id to enrich (required)")
	workflowsRunCmd.Flags().String("idempotency-key", "", "Idempotency key to safely retry run creation")
	workflowsRunsCmd.Flags().String("workflow-id", "", "Filter runs by workflow id")
	workflowsRunsCmd.Flags().Int("limit", 20, "Max runs to return")
	workflowsRunsCmd.Flags().Int("offset", 0, "Pagination offset")
	workflowsCancelCmd.Flags().Bool("yes", false, "Skip confirmation")
	workflowsDownloadCmd.Flags().String("out", "", "Output file path (default run-<id>.csv)")

	workflowsCmd.AddCommand(
		workflowsUploadCmd,
		workflowsRunCmd,
		workflowsStatusCmd,
		workflowsRunsCmd,
		workflowsCancelCmd,
		workflowsDownloadCmd,
		integrationsCmd,
	)
}
