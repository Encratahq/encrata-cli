package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

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
