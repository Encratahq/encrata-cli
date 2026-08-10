package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var jobsCancelCmd = &cobra.Command{
	Use:   "cancel [job-id]",
	Short: "Cancel a running job (use --type for identity or password)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jt, err := jobType(cmd)
		if err != nil {
			return err
		}
		if jt != "validity" {
			return cancelNonValidityJob(cmd, args, jt)
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Cancelling job...")
		data, err := client.CancelValidityJob(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Job cancelled: " + args[0])
		return nil
	},
}

var jobsRetryCmd = &cobra.Command{
	Use:   "retry [job-id]",
	Short: "Retry dead-lettered chunks of a job (validity or identity)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jt, err := jobType(cmd)
		if err != nil {
			return err
		}
		if jt != "validity" {
			return retryNonValidityJob(cmd, args, jt)
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Retrying job...")
		data, err := client.RetryValidityJob(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}

		requeued := 0
		var m map[string]interface{}
		if json.Unmarshal(data, &m) == nil {
			requeued = getInt(m, "requeued")
		}
		output.SuccessMsg(fmt.Sprintf("Requeued %d %s", requeued, plural(requeued, "chunk", "chunks")))

		statusData, err := client.GetValidityJob(cmd.Context(), args[0])
		if err != nil {
			return nil
		}
		job, err := api.ParseJob(statusData)
		if err != nil {
			return nil
		}
		output.Header("Validity Job: " + args[0])
		printJob(job)
		return nil
	},
}
