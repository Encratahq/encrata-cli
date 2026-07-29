package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var identityJobsCmd = &cobra.Command{
	Use:   "identity-jobs",
	Short: "Manage async identity enrichment jobs",
	Long: `Create and manage async identity enrichment jobs.

Examples:
  encrata identity-jobs create emails.csv
  encrata identity-jobs list
  encrata identity-jobs status <job-id>
  encrata identity-jobs results <job-id> --found-only
  encrata identity-jobs download <job-id> --out people.csv`,
}

var identityJobsCreateCmd = &cobra.Command{
	Use:   "create [file.csv]",
	Short: "Create an identity job from a file or STDIN",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := ""
		if len(args) == 1 {
			path = args[0]
		}
		fileName, emails, _, err := loadEmails(cmd, path)
		if err != nil {
			return err
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Creating identity job...")
		data, err := client.CreateIdentityJob(cmd.Context(), emails, fileName)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var job map[string]interface{}
		if !decode(data, &job) {
			return nil
		}
		output.Header("Identity Job Created")
		printIdentityJob(job)
		return nil
	},
}

var identityJobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List identity jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading jobs...")
		data, err := client.ListIdentityJobs(cmd.Context())
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
		output.Header(fmt.Sprintf("Identity Jobs: %d", len(jobs)))
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
				fmt.Sprintf("%d", getInt(m, "found_count")),
				fmt.Sprintf("%d", getInt(m, "credits_used")),
			})
		}
		output.Table([]string{"ID", "Status", "Total", "Processed", "Found", "Credits"}, rows)
		return nil
	},
}

var identityJobsStatusCmd = &cobra.Command{
	Use:   "status [job-id]",
	Short: "Show the status of an identity job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading job...")
		data, err := client.GetIdentityJob(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var job map[string]interface{}
		if !decode(data, &job) {
			return nil
		}
		output.Header("Identity Job: " + args[0])
		printIdentityJob(job)
		return nil
	},
}

var identityJobsResultsCmd = &cobra.Command{
	Use:   "results [job-id]",
	Short: "Fetch results of an identity job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")
		foundOnly, _ := cmd.Flags().GetBool("found-only")

		spinner := startSpinner("Loading results...")
		data, err := client.GetIdentityJobResults(cmd.Context(), args[0], page, pageSize, foundOnly)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		items := unwrapArray(data, "items")
		output.Header(fmt.Sprintf("Results: %d", len(items)))
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				firstNonEmpty(getStr(m, "email"), "—"),
				yesNo(boolField(m, "found")),
				firstNonEmpty(getStr(m, "name"), "—"),
				firstNonEmpty(getStr(m, "company"), "—"),
				firstNonEmpty(getStr(m, "job_role"), "—"),
				firstNonEmpty(getStr(m, "location"), "—"),
			})
		}
		output.Table([]string{"Email", "Found", "Name", "Company", "Role", "Location"}, rows)
		return nil
	},
}

var identityJobsDownloadCmd = &cobra.Command{
	Use:   "download [job-id]",
	Short: "Download identity job results as CSV",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		foundOnly, _ := cmd.Flags().GetBool("found-only")
		out, _ := cmd.Flags().GetString("out")

		spinner := startSpinner("Downloading results...")
		blob, err := client.DownloadIdentityJob(cmd.Context(), args[0], foundOnly)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
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

var identityJobsCancelCmd = &cobra.Command{
	Use:   "cancel [job-id]",
	Short: "Cancel a running identity job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Cancelling job...")
		data, err := client.CancelIdentityJob(cmd.Context(), args[0])
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

var identityJobsRetryCmd = &cobra.Command{
	Use:   "retry [job-id]",
	Short: "Retry dead-lettered chunks of an identity job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Retrying job...")
		data, err := client.RetryIdentityJob(cmd.Context(), args[0])
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

		statusData, err := client.GetIdentityJob(cmd.Context(), args[0])
		if err != nil {
			return nil
		}
		var job map[string]interface{}
		if !decode(statusData, &job) {
			return nil
		}
		output.Header("Identity Job: " + args[0])
		printIdentityJob(job)
		return nil
	},
}

// printIdentityJob renders an identity job status block.
func printIdentityJob(m map[string]interface{}) {
	output.KV(
		"ID", getStr(m, "id"),
		"Status", coloredStatus(getStr(m, "status")),
		"Total", fmt.Sprintf("%d", getInt(m, "total_emails")),
		"Processed", fmt.Sprintf("%d", getInt(m, "processed_count")),
		"Found", fmt.Sprintf("%d", getInt(m, "found_count")),
		"Credits used", fmt.Sprintf("%d", getInt(m, "credits_used")),
		"Created", jobCreated(getStr(m, "created_at")),
	)
}

func init() {
	// Superseded by `jobs <verb> --type identity`; kept working but hidden.
	identityJobsCmd.Hidden = true

	identityJobsResultsCmd.Flags().Int("page", 1, "Result page to fetch")
	identityJobsResultsCmd.Flags().Int("page-size", 50, "Results page size")
	identityJobsResultsCmd.Flags().Bool("found-only", false, "Only rows with enrichment data")

	identityJobsDownloadCmd.Flags().Bool("found-only", false, "Only rows with enrichment data")
	identityJobsDownloadCmd.Flags().String("out", "", "Write to a file instead of stdout")

	identityJobsCmd.AddCommand(
		identityJobsCreateCmd,
		identityJobsListCmd,
		identityJobsStatusCmd,
		identityJobsResultsCmd,
		identityJobsDownloadCmd,
		identityJobsCancelCmd,
		identityJobsRetryCmd,
	)
}
