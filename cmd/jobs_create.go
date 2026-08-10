package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var jobsCreateCmd = &cobra.Command{
	Use:   "create [file.csv]",
	Short: "Create an async job (validity, identity, or password) with --type",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jt, err := jobType(cmd)
		if err != nil {
			return err
		}
		if jt != "validity" {
			return createNonValidityJob(cmd, args, jt)
		}
		path := ""
		if len(args) == 1 {
			path = args[0]
		}
		fileName, _, raw, err := loadEmails(cmd, path)
		if err != nil {
			return err
		}

		client, err := newClient()
		if err != nil {
			return err
		}

		spinner := startSpinner("Creating validity job...")
		data, err := client.CreateValidityJobFile(cmd.Context(), fileName, raw)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}

		if jsonMode() {
			output.JSON(data)
			return nil
		}

		job, err := api.ParseJob(data)
		if err != nil {
			output.JSON(data)
			return nil
		}
		output.Header("Validity Job Created")
		printJob(job)
		return nil
	},
}

var jobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs (use --type for identity or password)",
	RunE: func(cmd *cobra.Command, args []string) error {
		jt, err := jobType(cmd)
		if err != nil {
			return err
		}
		if jt != "validity" {
			return listNonValidityJobs(cmd, jt)
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading jobs...")
		data, err := client.ListValidityJobs(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}

		jobs := unwrapArray(data, "jobs")
		output.Header(fmt.Sprintf("Validity Jobs: %d", len(jobs)))
		rows := make([][]string, 0, len(jobs))
		for _, item := range jobs {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "status"),
				fmt.Sprintf("%d", getInt(m, "total_emails")),
				fmt.Sprintf("%d", getInt(m, "processed_count")),
				fmt.Sprintf("%d", getInt(m, "credits_used")),
			})
		}
		output.Table([]string{"ID", "Status", "Total", "Processed", "Credits"}, rows)
		return nil
	},
}

var jobsStatusCmd = &cobra.Command{
	Use:   "status [job-id]",
	Short: "Show the status of a job (use --type for identity or password)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jt, err := jobType(cmd)
		if err != nil {
			return err
		}
		if jt != "validity" {
			return statusNonValidityJob(cmd, args, jt)
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading job...")
		data, err := client.GetValidityJob(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		job, err := api.ParseJob(data)
		if err != nil {
			output.JSON(data)
			return nil
		}
		output.Header("Validity Job: " + args[0])
		printJob(job)
		return nil
	},
}
