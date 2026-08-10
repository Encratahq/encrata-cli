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
		// Download the raw result objects and flatten them client-side so job
		// exports use the same column set and formats as streaming exports.
		spinner := startSpinner("Downloading results...")
		blob, err := client.DownloadValidityJob(cmd.Context(), final.ID, "", "json")
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		raw := unwrapArray(json.RawMessage(blob), "results")
		results := make([]map[string]interface{}, 0, len(raw))
		for _, item := range raw {
			if m, ok := item.(map[string]interface{}); ok {
				results = append(results, m)
			}
		}
		return exportBulk(cmd, out, results)
	}
	return nil
}
