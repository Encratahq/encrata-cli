package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// decodeRow unmarshals a single JSON object into a map.
func decodeRow(data json.RawMessage) (map[string]interface{}, bool) {
	var m map[string]interface{}
	if json.Unmarshal(data, &m) != nil {
		return nil, false
	}
	return m, true
}

func runBulkJob(cmd *cobra.Command, client api.API, fileName string, raw []byte, out string) error {
	spinner := startSpinner("Creating validity job...")
	data, err := client.CreateValidityJobFile(cmd.Context(), fileName, raw)
	stopSpinner(spinner)
	if err != nil {
		output.Error(err.Error())
		return err
	}

	job, err := api.ParseJob(data)
	if err != nil || job.ID == "" {
		output.JSON(data)
		return err
	}

	asJSON := jsonMode()
	if !asJSON {
		output.Header("Validity Job: " + job.ID)
	}

	final, err := client.PollValidityJob(cmd.Context(), job.ID, 2*time.Second, func(j *api.Job) {
		if !asJSON && j.TotalEmails > 0 {
			renderProgress(j.ProcessedCount, j.TotalEmails)
		}
	})
	if !asJSON {
		fmt.Println()
	}
	if err != nil {
		output.Error(err.Error())
		return err
	}

	printJob(final)

	if out != "" {
		// Server-built cached export: the file is streamed into S3 once and pulled
		// from a presigned URL, so a 1M-row job never loads every row into the CLI.
		format, ferr := resolveExportFormat(flagString(cmd, "format"), out)
		if ferr != nil {
			return friendlyFormatError(cmd, ferr.Error())
		}
		if err := runJobExport(cmd.Context(), client, "validity", final.ID, out, format, validityExportFilter(cmd)); err != nil {
			return err
		}
		printLeanExportHint(cmd, out)
		return nil
	}
	return nil
}
