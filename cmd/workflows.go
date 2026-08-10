package cmd

import (
	"strconv"

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
