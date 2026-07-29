package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Manage async jobs (validity, identity, password)",
}

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

// printJob renders a job status block with counts, credits, and created time.
func printJob(job *api.Job) {
	status := job.Status
	if colored := statusColor(job.Status); colored != "" {
		status = colored
	}
	output.KV(
		"ID", job.ID,
		"Status", status,
		"Total", fmt.Sprintf("%d", job.TotalEmails),
		"Processed", fmt.Sprintf("%d", job.ProcessedCount),
		"Valid", fmt.Sprintf("%d", job.SuccessCount),
		"Errors", fmt.Sprintf("%d", job.ErrorCount),
		"Credits used", fmt.Sprintf("%d", job.CreditsUsed),
		"Created", jobCreated(job.CreatedAt),
	)
}

// jobCreated renders a job timestamp in local, human-readable form.
func jobCreated(ts string) string {
	if ts == "" {
		return ""
	}
	return timeField(map[string]interface{}{"t": ts}, "t")
}

// coloredStatus returns a color-themed status label, falling back to the raw
// string when no theme is defined for it.
func coloredStatus(s string) string {
	if c := statusColor(s); c != "" {
		return c
	}
	return s
}

func filterRowsByStatus(rows []map[string]interface{}, status string) []map[string]interface{} {
	want := strings.ToLower(strings.TrimSpace(status))
	if want == "" {
		return rows
	}
	filtered := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		raw := strings.ToLower(strings.TrimSpace(firstNonEmpty(getStr(row, "status"), getStr(row, "validity"))))
		normalized := normalizedValidityStatus(raw)
		if raw == want || normalized == want {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func init() {
	// --type selects the job kind on every unified command (default validity).
	for _, c := range []*cobra.Command{
		jobsCreateCmd, jobsListCmd, jobsStatusCmd, jobsResultsCmd,
		jobsDownloadCmd, jobsCancelCmd, jobsRetryCmd,
	} {
		c.Flags().String("type", "validity", "Job type: validity | identity | password")
	}

	// create: identity/validity read emails from the file arg or STDIN;
	// password reads hashes from these flags.
	jobsCreateCmd.Flags().StringSlice("sha1s", nil, "Password: inline SHA-1 hashes (40-char hex)")
	jobsCreateCmd.Flags().String("sha1-file", "", "Password: read SHA-1 hashes from a file")
	jobsCreateCmd.Flags().String("password-file", "", "Password: read plaintext passwords from a file (hashed locally)")
	jobsCreateCmd.Flags().String("file-name", "", "Optional display name for the job")

	jobsResultsCmd.Flags().String("status", "", "Filter results by per-row status (e.g. valid, invalid)")
	jobsResultsCmd.Flags().Int("page", 1, "Result page to fetch")
	jobsResultsCmd.Flags().Int("page-size", 50, "Results per page (identity/password)")
	jobsResultsCmd.Flags().Bool("found-only", false, "Identity: only rows with enrichment data")
	jobsResultsCmd.Flags().Bool("breached", false, "Password: only breached rows")
	jobsResultsCmd.Flags().StringSlice("fields", nil, "Extra columns from the validity schema (e.g. provider,mx)")

	jobsDownloadCmd.Flags().String("format", "csv", "Download format: csv, xlsx, or json (validity)")
	jobsDownloadCmd.Flags().String("status", "", "Filter rows by status")
	jobsDownloadCmd.Flags().Bool("valid-only", false, "Download only rows whose status is valid")
	jobsDownloadCmd.Flags().Bool("found-only", false, "Identity: only rows with enrichment data")
	jobsDownloadCmd.Flags().Bool("breached", false, "Password: only breached rows")
	jobsDownloadCmd.Flags().String("out", "", "Write to a file instead of stdout")

	jobsCmd.AddCommand(
		jobsCreateCmd,
		jobsListCmd,
		jobsStatusCmd,
		jobsResultsCmd,
		jobsDownloadCmd,
		jobsCancelCmd,
		jobsRetryCmd,
	)
}
