package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/validation"
	"github.com/spf13/cobra"
)

var sha1Regex = regexp.MustCompile(`^[A-F0-9]{40}$`)

var jobsBulkValidateCmd = &cobra.Command{
	Use:   "bulk-validate-emails",
	Short: "Create a validity async job from inline emails",
	RunE: func(cmd *cobra.Command, args []string) error {
		emails, err := gatherJobEmails(cmd)
		if err != nil {
			return err
		}
		if len(emails) == 0 {
			return friendlyFormatError(cmd, "provide at least one email via --emails or --file")
		}
		fileName, _ := cmd.Flags().GetString("file-name")
		batchID, _ := cmd.Flags().GetString("batch-id")

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Creating validity job...")
		data, err := client.CreateValidityJob(cmd.Context(), &api.CreateJobRequest{Emails: emails, FileName: fileName, BatchID: batchID})
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		return renderUnifiedJSONResult("Validity Job Created", data)
	},
}

var jobsBulkIdentityCmd = &cobra.Command{
	Use:   "bulk-email-identity",
	Short: "Create an identity async job from inline emails",
	RunE: func(cmd *cobra.Command, args []string) error {
		emails, err := gatherJobEmails(cmd)
		if err != nil {
			return err
		}
		if len(emails) == 0 {
			return friendlyFormatError(cmd, "provide at least one email via --emails or --file")
		}
		fileName, _ := cmd.Flags().GetString("file-name")

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
		return renderUnifiedJSONResult("Identity Job Created", data)
	},
}

var jobsBulkPasswordCmd = &cobra.Command{
	Use:   "bulk-password-breaches",
	Short: "Create a password breach async job from SHA-1 hashes",
	RunE: func(cmd *cobra.Command, args []string) error {
		sha1s, err := gatherSHA1s(cmd)
		if err != nil {
			return err
		}
		if len(sha1s) == 0 {
			return friendlyFormatError(cmd, "provide at least one SHA-1 via --sha1s or --sha1-file")
		}
		fileName, _ := cmd.Flags().GetString("file-name")

		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Creating password job...")
		data, err := client.CreatePasswordJob(cmd.Context(), sha1s, fileName)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		return renderUnifiedJSONResult("Password Job Created", data)
	},
}

var jobsGetStatusCmd = &cobra.Command{
	Use:   "get-email-job-status [job-id]",
	Short: "Get validity/identity/password job status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobType, err := jobTypeFlag(cmd)
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}

		spinner := startSpinner("Loading job status...")
		data, err := fetchJobStatus(cmd, client, jobType, args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		return renderUnifiedJSONResult(jobTypeTitle(jobType)+" Job Status", data)
	},
}

var jobsGetResultsCmd = &cobra.Command{
	Use:   "get-email-job-results [job-id]",
	Short: "Get paginated results for validity/identity/password jobs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobType, err := jobTypeFlag(cmd)
		if err != nil {
			return err
		}
		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")
		breached, _ := cmd.Flags().GetBool("breached")
		foundOnly, _ := cmd.Flags().GetBool("found-only")

		client, err := newClient()
		if err != nil {
			return err
		}

		spinner := startSpinner("Loading job results...")
		data, err := fetchJobResults(cmd, client, jobType, args[0], page, pageSize, breached, foundOnly)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		return renderUnifiedJSONResult(jobTypeTitle(jobType)+" Job Results", data)
	},
}

var jobsDownloadEmailCmd = &cobra.Command{
	Use:   "download-email-job [job-id]",
	Short: "Download CSV or JSON for validity/identity/password jobs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobType, err := jobTypeFlag(cmd)
		if err != nil {
			return err
		}
		format, _ := cmd.Flags().GetString("format")
		if format != "csv" {
			return friendlyFormatError(cmd, "format must be csv")
		}
		breached, _ := cmd.Flags().GetBool("breached")
		foundOnly, _ := cmd.Flags().GetBool("found-only")
		out, _ := cmd.Flags().GetString("out")
		status, _ := cmd.Flags().GetString("status")

		client, err := newClient()
		if err != nil {
			return err
		}

		spinner := startSpinner("Downloading job results...")
		blob, err := downloadJob(cmd, client, jobType, args[0], format, status, breached, foundOnly)
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

var jobsCancelEmailCmd = &cobra.Command{
	Use:   "cancel-email-job [job-id]",
	Short: "Cancel a validity/identity/password job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobType, err := jobTypeFlag(cmd)
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}

		spinner := startSpinner("Cancelling job...")
		data, err := cancelJob(cmd, client, jobType, args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Cancelled " + jobType + " job: " + args[0])
		return nil
	},
}

var jobsRetryEmailCmd = &cobra.Command{
	Use:   "retry-email-job [job-id]",
	Short: "Retry dead-lettered chunks for a job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobType, err := jobTypeFlag(cmd)
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}

		spinner := startSpinner("Retrying job...")
		data, err := retryJob(cmd, client, jobType, args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		return renderUnifiedJSONResult(jobTypeTitle(jobType)+" Job Retry", data)
	},
}

var jobsListBulkCmd = &cobra.Command{
	Use:   "list-bulk-jobs",
	Short: "List all async bulk jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading bulk jobs...")
		data, err := client.ListBulkJobs(cmd.Context())
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		return renderUnifiedJSONResult("Bulk Jobs", data)
	},
}

var jobsGetBulkCmd = &cobra.Command{
	Use:   "get-bulk-job [job-id]",
	Short: "Get a bulk job by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading bulk job...")
		data, err := client.GetBulkJob(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		return renderUnifiedJSONResult("Bulk Job", data)
	},
}

var jobsCancelBulkCmd = &cobra.Command{
	Use:   "cancel-bulk-job [job-id]",
	Short: "Cancel a bulk job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Cancelling bulk job...")
		data, err := client.CancelBulkJob(cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		output.SuccessMsg("Bulk job cancelled: " + args[0])
		return nil
	},
}

func gatherJobEmails(cmd *cobra.Command) ([]string, error) {
	emails, _ := cmd.Flags().GetStringSlice("emails")
	filePath, _ := cmd.Flags().GetString("file")

	seen := map[string]bool{}
	out := make([]string, 0)
	appendEmail := func(email string) error {
		email = strings.TrimSpace(email)
		if email == "" {
			return nil
		}
		if err := validation.Email(email); err != nil {
			return friendlyFormatError(cmd, err.Error())
		}
		key := strings.ToLower(email)
		if !seen[key] {
			seen[key] = true
			out = append(out, email)
		}
		return nil
	}

	for _, email := range emails {
		if err := appendEmail(email); err != nil {
			return nil, err
		}
	}

	if filePath != "" {
		raw, err := readFileBytes(filePath)
		if err != nil {
			return nil, err
		}
		for _, email := range parseEmails(raw) {
			if err := appendEmail(email); err != nil {
				return nil, err
			}
		}
	}

	return out, nil
}

func gatherSHA1s(cmd *cobra.Command) ([]string, error) {
	sha1s, _ := cmd.Flags().GetStringSlice("sha1s")
	filePath, _ := cmd.Flags().GetString("sha1-file")

	seen := map[string]bool{}
	out := make([]string, 0)
	appendHash := func(hash string) error {
		hash = strings.ToUpper(strings.TrimSpace(hash))
		if hash == "" {
			return nil
		}
		if !sha1Regex.MatchString(hash) {
			return friendlyFormatError(cmd, "invalid SHA-1 hash; expected 40 hex characters")
		}
		if !seen[hash] {
			seen[hash] = true
			out = append(out, hash)
		}
		return nil
	}

	for _, hash := range sha1s {
		if err := appendHash(hash); err != nil {
			return nil, err
		}
	}
	if filePath != "" {
		lines, err := readLines(filePath)
		if err != nil {
			return nil, err
		}
		for _, hash := range lines {
			if err := appendHash(hash); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

func jobTypeFlag(cmd *cobra.Command) (string, error) {
	jobType, _ := cmd.Flags().GetString("job-type")
	jobType = strings.ToLower(strings.TrimSpace(jobType))
	switch jobType {
	case "validity", "identity", "password":
		return jobType, nil
	default:
		return "", friendlyFormatError(cmd, "job-type must be validity, identity, or password")
	}
}

func fetchJobStatus(cmd *cobra.Command, client *api.Client, jobType, jobID string) (json.RawMessage, error) {
	switch jobType {
	case "validity":
		return client.GetValidityJob(cmd.Context(), jobID)
	case "identity":
		return client.GetIdentityJob(cmd.Context(), jobID)
	default:
		return client.GetPasswordJob(cmd.Context(), jobID)
	}
}

func fetchJobResults(cmd *cobra.Command, client *api.Client, jobType, jobID string, page, pageSize int, breached, foundOnly bool) (json.RawMessage, error) {
	switch jobType {
	case "validity":
		return client.GetValidityJobResultsPage(cmd.Context(), jobID, page, pageSize)
	case "identity":
		return client.GetIdentityJobResults(cmd.Context(), jobID, page, pageSize, foundOnly || breached)
	default:
		return client.GetPasswordJobResults(cmd.Context(), jobID, page, pageSize, breached)
	}
}

func downloadJob(cmd *cobra.Command, client *api.Client, jobType, jobID, format, status string, breached, foundOnly bool) ([]byte, error) {
	switch jobType {
	case "validity":
		return client.DownloadValidityJob(cmd.Context(), jobID, status, format)
	case "identity":
		return client.DownloadIdentityJob(cmd.Context(), jobID, foundOnly || breached)
	default:
		return client.DownloadPasswordJob(cmd.Context(), jobID, breached)
	}
}

func cancelJob(cmd *cobra.Command, client *api.Client, jobType, jobID string) (json.RawMessage, error) {
	switch jobType {
	case "validity":
		return client.CancelValidityJob(cmd.Context(), jobID)
	case "identity":
		return client.CancelIdentityJob(cmd.Context(), jobID)
	default:
		return client.CancelPasswordJob(cmd.Context(), jobID)
	}
}

func retryJob(cmd *cobra.Command, client *api.Client, jobType, jobID string) (json.RawMessage, error) {
	switch jobType {
	case "validity":
		return client.RetryValidityJob(cmd.Context(), jobID)
	case "identity":
		return client.RetryIdentityJob(cmd.Context(), jobID)
	default:
		return client.RetryPasswordJob(cmd.Context(), jobID)
	}
}

func renderUnifiedJSONResult(title string, data json.RawMessage) error {
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	output.Header(title)
	output.JSON(data)
	return nil
}

func jobTypeTitle(jobType string) string {
	switch jobType {
	case "identity":
		return "Identity"
	case "password":
		return "Password"
	default:
		return "Validity"
	}
}

func init() {
	for _, command := range []*cobra.Command{jobsBulkValidateCmd, jobsBulkIdentityCmd} {
		command.Flags().StringSlice("emails", nil, "Inline emails")
		command.Flags().String("file", "", "Read emails from a file")
		command.Flags().String("file-name", "", "Optional file name label")
	}
	jobsBulkValidateCmd.Flags().String("batch-id", "", "Optional grouping ID")

	jobsBulkPasswordCmd.Flags().StringSlice("sha1s", nil, "Inline SHA-1 hashes (40-char hex)")
	jobsBulkPasswordCmd.Flags().String("sha1-file", "", "Read SHA-1 hashes from a file")
	jobsBulkPasswordCmd.Flags().String("file-name", "", "Optional file name label")

	for _, command := range []*cobra.Command{jobsGetStatusCmd, jobsGetResultsCmd, jobsDownloadEmailCmd, jobsCancelEmailCmd, jobsRetryEmailCmd} {
		command.Flags().String("job-type", "validity", "Job type: validity | identity | password")
	}

	jobsGetResultsCmd.Flags().Int("page", 1, "1-based page number")
	jobsGetResultsCmd.Flags().Int("page-size", 50, "Results page size")
	jobsGetResultsCmd.Flags().Bool("breached", false, "Password: breached only; Identity: found only")
	jobsGetResultsCmd.Flags().Bool("found-only", false, "Identity results only")

	jobsDownloadEmailCmd.Flags().String("format", "csv", "Download format: csv")
	jobsDownloadEmailCmd.Flags().String("status", "", "Validity-only status filter")
	jobsDownloadEmailCmd.Flags().Bool("breached", false, "Password: breached only; Identity: found only")
	jobsDownloadEmailCmd.Flags().Bool("found-only", false, "Identity: found only")
	jobsDownloadEmailCmd.Flags().String("out", "", "Write output to a file")

	// These MCP-style commands are superseded by the unified `jobs <verb> --type`
	// surface. They remain functional (for MCP parity and existing scripts) but
	// are hidden from help.
	deprecated := []*cobra.Command{
		jobsBulkValidateCmd,
		jobsBulkIdentityCmd,
		jobsBulkPasswordCmd,
		jobsGetStatusCmd,
		jobsGetResultsCmd,
		jobsDownloadEmailCmd,
		jobsCancelEmailCmd,
		jobsRetryEmailCmd,
		jobsListBulkCmd,
		jobsGetBulkCmd,
		jobsCancelBulkCmd,
	}
	for _, c := range deprecated {
		c.Hidden = true
		jobsCmd.AddCommand(c)
	}
}
