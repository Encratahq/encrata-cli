package cmd

import (
	"encoding/json"
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
		status, _ := cmd.Flags().GetString("status")
		validOnly, _ := cmd.Flags().GetBool("valid-only")
		out, _ := cmd.Flags().GetString("out")

		if format != "csv" && format != "json" && format != "xlsx" {
			return friendlyFormatError(cmd, "format must be csv, xlsx, or json")
		}
		if validOnly {
			status = "valid"
		}

		if out == "" && (format == "csv" || format == "xlsx") {
			out = defaultValidityDownloadName(format)
		}

		spinner := startSpinner("Downloading results...")
		apiFormat := format
		if format == "xlsx" || format == "csv" {
			apiFormat = "json"
		}
		blob, err := client.DownloadValidityJob(cmd.Context(), args[0], status, apiFormat)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}

		if format == "xlsx" || format == "csv" {
			raw := unwrapArray(json.RawMessage(blob), "results")
			rows := make([]map[string]interface{}, 0, len(raw))
			for _, item := range raw {
				if m, ok := item.(map[string]interface{}); ok {
					rows = append(rows, m)
				}
			}
			rows = filterRowsByStatus(rows, status)

			if format == "xlsx" {
				if out == "" {
					out = defaultValidityDownloadName("xlsx")
				}
				if err := writeXLSX(out, selectExportColumns(nil), rows); err != nil {
					return err
				}
				output.SuccessMsg(fmt.Sprintf("Wrote %d %s to %s", len(rows), plural(len(rows), "row", "rows"), out))
				return nil
			}

			csvBlob, err := buildFlatCSV(selectExportColumns(nil), rows)
			if err != nil {
				return err
			}
			if out == "" {
				fmt.Print(string(csvBlob))
				return nil
			}
			if err := writeFileBytes(out, csvBlob); err != nil {
				return err
			}
			output.SuccessMsg(fmt.Sprintf("Wrote %d %s to %s", len(rows), plural(len(rows), "row", "rows"), out))
			return nil
		}

		if out == "" {
			fmt.Print(string(blob))
			return nil
		}
		if err := writeFileBytes(out, blob); err != nil {
			return err
		}
		output.SuccessMsg("Wrote results to " + out)
		return nil
	},
}
