package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/password"
	"github.com/spf13/cobra"
)

var passwordJobsCmd = &cobra.Command{
	Use:   "password-jobs",
	Short: "Manage async password breach jobs",
	Long: `Create and manage async password-breach jobs.

Passwords are hashed locally (SHA-1) — plaintext never leaves your machine.
Provide hashes directly with --sha1s/--sha1-file, or a plaintext list with
--password-file (hashed locally before upload).

Examples:
  encrata password-jobs create --sha1-file hashes.txt
  encrata password-jobs create --password-file passwords.txt
  encrata password-jobs list
  encrata password-jobs results <job-id> --breached-only`,
}

var passwordJobsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a password breach job from SHA-1 hashes",
	RunE: func(cmd *cobra.Command, args []string) error {
		sha1s, err := gatherPasswordSHA1s(cmd)
		if err != nil {
			return err
		}
		if len(sha1s) == 0 {
			return friendlyFormatError(cmd, "provide hashes via --sha1s/--sha1-file or plaintext via --password-file")
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
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var job map[string]interface{}
		if !decode(data, &job) {
			return nil
		}
		output.Header("Password Job Created")
		printPasswordJob(job)
		return nil
	},
}

var passwordJobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List password jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading jobs...")
		data, err := client.ListPasswordJobs(cmd.Context())
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
		output.Header(fmt.Sprintf("Password Jobs: %d", len(jobs)))
		rows := make([][]string, 0, len(jobs))
		for _, item := range jobs {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "status"),
				fmt.Sprintf("%d", getInt(m, "total")),
				fmt.Sprintf("%d", getInt(m, "processed_count")),
				fmt.Sprintf("%d", getInt(m, "breached_count")),
				fmt.Sprintf("%d", getInt(m, "credits_used")),
			})
		}
		output.Table([]string{"ID", "Status", "Total", "Processed", "Breached", "Credits"}, rows)
		return nil
	},
}

var passwordJobsStatusCmd = &cobra.Command{
	Use:   "status [job-id]",
	Short: "Show the status of a password job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading job...")
		data, err := client.GetPasswordJob(cmd.Context(), args[0])
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
		output.Header("Password Job: " + args[0])
		printPasswordJob(job)
		return nil
	},
}

var passwordJobsResultsCmd = &cobra.Command{
	Use:   "results [job-id]",
	Short: "Fetch results of a password job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")
		breachedOnly, _ := cmd.Flags().GetBool("breached-only")

		spinner := startSpinner("Loading results...")
		data, err := client.GetPasswordJobResults(cmd.Context(), args[0], page, pageSize, breachedOnly)
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
				fmt.Sprintf("%d", getInt(m, "line_no")),
				firstNonEmpty(getStr(m, "prefix"), "—"),
				yesNo(boolField(m, "found")),
				fmt.Sprintf("%d", getInt(m, "count")),
			})
		}
		output.Table([]string{"Line", "Prefix", "Breached", "Count"}, rows)
		return nil
	},
}

var passwordJobsDownloadCmd = &cobra.Command{
	Use:   "download [job-id]",
	Short: "Download password job results as CSV",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		breachedOnly, _ := cmd.Flags().GetBool("breached-only")
		out, _ := cmd.Flags().GetString("out")

		spinner := startSpinner("Downloading results...")
		blob, err := client.DownloadPasswordJob(cmd.Context(), args[0], breachedOnly)
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

var passwordJobsCancelCmd = &cobra.Command{
	Use:   "cancel [job-id]",
	Short: "Cancel a running password job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Cancelling job...")
		data, err := client.CancelPasswordJob(cmd.Context(), args[0])
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

// gatherPasswordSHA1s collects SHA-1 hashes from --sha1s/--sha1-file plus any
// plaintext passwords from --password-file (hashed locally, never uploaded).
func gatherPasswordSHA1s(cmd *cobra.Command) ([]string, error) {
	out, err := gatherSHA1s(cmd)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(out))
	for _, h := range out {
		seen[h] = true
	}

	pwFile, _ := cmd.Flags().GetString("password-file")
	if pwFile != "" {
		lines, err := readLines(pwFile)
		if err != nil {
			return nil, err
		}
		for _, line := range lines {
			buf := []byte(line)
			h := password.Hash(buf)
			password.Zero(buf)
			if !seen[h] {
				seen[h] = true
				out = append(out, h)
			}
		}
	}
	return out, nil
}

// printPasswordJob renders a password job status block.
func printPasswordJob(m map[string]interface{}) {
	output.KV(
		"ID", getStr(m, "id"),
		"Status", coloredStatus(getStr(m, "status")),
		"Total", fmt.Sprintf("%d", getInt(m, "total")),
		"Processed", fmt.Sprintf("%d", getInt(m, "processed_count")),
		"Breached", fmt.Sprintf("%d", getInt(m, "breached_count")),
		"Credits used", fmt.Sprintf("%d", getInt(m, "credits_used")),
		"Created", jobCreated(getStr(m, "created_at")),
	)
}

func init() {
	// Superseded by `jobs <verb> --type password`; kept working but hidden.
	passwordJobsCmd.Hidden = true

	passwordJobsCreateCmd.Flags().StringSlice("sha1s", nil, "Inline SHA-1 hashes (40-char hex)")
	passwordJobsCreateCmd.Flags().String("sha1-file", "", "Read SHA-1 hashes from a file (one per line)")
	passwordJobsCreateCmd.Flags().String("password-file", "", "Read plaintext passwords from a file (hashed locally)")
	passwordJobsCreateCmd.Flags().String("file-name", "", "Optional file name label")

	passwordJobsResultsCmd.Flags().Int("page", 1, "Result page to fetch")
	passwordJobsResultsCmd.Flags().Int("page-size", 50, "Results page size")
	passwordJobsResultsCmd.Flags().Bool("breached-only", false, "Only rows found in a breach")

	passwordJobsDownloadCmd.Flags().Bool("breached-only", false, "Only rows found in a breach")
	passwordJobsDownloadCmd.Flags().String("out", "", "Write to a file instead of stdout")

	passwordJobsCmd.AddCommand(
		passwordJobsCreateCmd,
		passwordJobsListCmd,
		passwordJobsStatusCmd,
		passwordJobsResultsCmd,
		passwordJobsDownloadCmd,
		passwordJobsCancelCmd,
	)
}
