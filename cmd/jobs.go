package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Manage async bulk jobs",
}

var jobsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an async bulk email job",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		emails, err := collectInputs(cmd, args)
		if err != nil {
			return err
		}

		data, err := client.CreateBulkJob(cmd.Context(), emails)
		if err != nil {
			return err
		}

		if jsonMode() {
			output.JSON(data)
			return nil
		}

		var result map[string]interface{}
		if !decode(data, &result) {
			return nil
		}

		output.Header("Bulk Job Created")
		output.KV(
			"ID", getStr(result, "id"),
			"Status", getStr(result, "status"),
			"Emails", fmt.Sprintf("%d", getInt(result, "total_emails")),
		)
		return nil
	},
}

var jobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List async bulk jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		data, err := client.ListBulkJobs(cmd.Context())
		if err != nil {
			return err
		}

		if jsonMode() {
			output.JSON(data)
			return nil
		}

		var jobs []map[string]interface{}
		if !decode(data, &jobs) {
			return nil
		}

		output.Header(fmt.Sprintf("Bulk Jobs: %d", len(jobs)))
		for _, job := range jobs {
			output.Bold.Printf("  %s\n", getStr(job, "id"))
			output.KV(
				"Status", getStr(job, "status"),
				"Total", fmt.Sprintf("%d", getInt(job, "total_emails")),
				"Processed", fmt.Sprintf("%d", getInt(job, "processed_count")),
				"Success", fmt.Sprintf("%d", getInt(job, "success_count")),
				"Errors", fmt.Sprintf("%d", getInt(job, "error_count")),
				"Credits", fmt.Sprintf("%d", getInt(job, "credits_used")),
			)
			fmt.Println()
		}
		return nil
	},
}

var jobsGetCmd = &cobra.Command{
	Use:   "get [job-id]",
	Short: "Get async bulk job status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		data, err := client.GetBulkJob(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		if jsonMode() {
			output.JSON(data)
			return nil
		}

		var resp struct {
			Job         map[string]interface{} `json:"job"`
			DownloadURL string                 `json:"download_url"`
		}
		if !decode(data, &resp) {
			return nil
		}

		output.Header("Bulk Job: " + args[0])
		output.KV(
			"Status", getStr(resp.Job, "status"),
			"Total", fmt.Sprintf("%d", getInt(resp.Job, "total_emails")),
			"Processed", fmt.Sprintf("%d", getInt(resp.Job, "processed_count")),
			"Success", fmt.Sprintf("%d", getInt(resp.Job, "success_count")),
			"Errors", fmt.Sprintf("%d", getInt(resp.Job, "error_count")),
			"Credits", fmt.Sprintf("%d", getInt(resp.Job, "credits_used")),
		)
		if resp.DownloadURL != "" {
			output.KV("Download", resp.DownloadURL)
		}
		return nil
	},
}

var jobsCancelCmd = &cobra.Command{
	Use:   "cancel [job-id]",
	Short: "Cancel an async bulk job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		data, err := client.CancelBulkJob(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		if jsonMode() {
			output.JSON(data)
			return nil
		}

		output.SuccessMsg("Bulk job cancelled.")
		return nil
	},
}

func init() {
	jobsCreateCmd.Flags().StringP("file", "f", "", "Read emails from a file")
	jobsCmd.AddCommand(jobsCreateCmd)
	jobsCmd.AddCommand(jobsListCmd)
	jobsCmd.AddCommand(jobsGetCmd)
	jobsCmd.AddCommand(jobsCancelCmd)
}
