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
