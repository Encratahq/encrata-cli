package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var jobsResultsCmd = &cobra.Command{
	Use:   "results [job-id]",
	Short: "Fetch results of a job (use --type for identity or password)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jt, err := jobType(cmd)
		if err != nil {
			return err
		}
		if jt != "validity" {
			return resultsNonValidityJob(cmd, args, jt)
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		status, _ := cmd.Flags().GetString("status")
		if status == "" {
			status = validityOnlyStatus(cmd)
		}
		page, _ := cmd.Flags().GetInt("page")

		spinner := startSpinner("Loading results...")
		data, err := client.GetValidityJobResults(cmd.Context(), args[0], page, status)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}

		fields, _ := cmd.Flags().GetStringSlice("fields")
		raw := unwrapArray(data, "results")
		results := make([]map[string]interface{}, 0, len(raw))
		for _, item := range raw {
			if m, ok := item.(map[string]interface{}); ok {
				results = append(results, m)
			}
		}
		output.Header(fmt.Sprintf("Results: %d", len(results)))
		printResultsTable(results, fields)
		return nil
	},
}

var jobsDownloadCmd = &cobra.Command{
	Use:   "download [job-id]",
	Short: "Download job results (use --type for identity or password)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jt, err := jobType(cmd)
		if err != nil {
			return err
		}
		if jt != "validity" {
			return downloadNonValidityJob(cmd, args, jt)
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		format, _ := cmd.Flags().GetString("format")
		out, _ := cmd.Flags().GetString("out")

		if format != "csv" && format != "json" && format != "xlsx" {
			return friendlyFormatError(cmd, "format must be csv, xlsx, or json")
		}
		if out == "" && (format == "csv" || format == "xlsx") {
			out = defaultResultPath(args[0], format)
		}

		// Server-built cached export (streamed from S3), replacing the old sync
		// streaming download that pulled every row into the CLI to flatten locally.
		return runJobExport(cmd.Context(), client, "validity", args[0], out, format, validityExportFilter(cmd))
	},
}
