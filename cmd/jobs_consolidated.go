package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// jobType reads and validates the --type flag on the unified `jobs` commands.
// Defaults to validity so existing (validity-only) invocations are unchanged.
func jobType(cmd *cobra.Command) (string, error) {
	t, _ := cmd.Flags().GetString("type")
	switch t {
	case "", "validity":
		return "validity", nil
	case "identity", "password":
		return t, nil
	default:
		return "", friendlyFormatError(cmd, "--type must be validity, identity, or password")
	}
}

// ── create ──────────────────────────────────────────────────────────────────

func createNonValidityJob(cmd *cobra.Command, args []string, jt string) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	fileName, _ := cmd.Flags().GetString("file-name")

	if jt == "password" {
		sha1s, err := gatherPasswordSHA1s(cmd)
		if err != nil {
			return err
		}
		if len(sha1s) == 0 {
			return friendlyFormatError(cmd, "provide hashes via --sha1s/--sha1-file or --password-file")
		}
		spinner := startSpinner("Creating password job...")
		data, err := client.CreatePasswordJob(cmd.Context(), sha1s, fileName)
		stopSpinner(spinner)
		if err != nil {
			return jobsError(err)
		}
		return renderCreatedJob(cmd, "password", data)
	}

	// identity — read emails from a file arg or STDIN
	path := ""
	if len(args) == 1 {
		path = args[0]
	}
	name, emails, _, err := loadEmails(cmd, path)
	if err != nil {
		return err
	}
	if fileName == "" {
		fileName = name
	}
	spinner := startSpinner("Creating identity job...")
	data, err := client.CreateIdentityJob(cmd.Context(), emails, fileName)
	stopSpinner(spinner)
	if err != nil {
		return jobsError(err)
	}
	return renderCreatedJob(cmd, "identity", data)
}

func renderCreatedJob(cmd *cobra.Command, jt string, data json.RawMessage) error {
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	var m map[string]interface{}
	if !decode(data, &m) {
		return nil
	}
	output.Header(jobTypeTitle(jt) + " Job Created")
	if jt == "password" {
		printPasswordJob(m)
	} else {
		printIdentityJob(m)
	}
	return nil
}

// ── list ────────────────────────────────────────────────────────────────────

func listNonValidityJobs(cmd *cobra.Command, jt string) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	spinner := startSpinner("Loading jobs...")
	var data json.RawMessage
	if jt == "password" {
		data, err = client.ListPasswordJobs(cmd.Context())
	} else {
		data, err = client.ListIdentityJobs(cmd.Context())
	}
	stopSpinner(spinner)
	if err != nil {
		return jobsError(err)
	}
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	jobs := unwrapArray(data, "jobs")
	output.Header(fmt.Sprintf("%s Jobs: %d", jobTypeTitle(jt), len(jobs)))
	rows := make([][]string, 0, len(jobs))
	for _, item := range jobs {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if jt == "password" {
			rows = append(rows, []string{
				getStr(m, "id"), getStr(m, "status"),
				fmt.Sprintf("%d", getInt(m, "total")),
				fmt.Sprintf("%d", getInt(m, "processed_count")),
				fmt.Sprintf("%d", getInt(m, "breached_count")),
				fmt.Sprintf("%d", getInt(m, "credits_used")),
			})
		} else {
			rows = append(rows, []string{
				getStr(m, "id"), getStr(m, "status"),
				fmt.Sprintf("%d", getInt(m, "total_emails")),
				fmt.Sprintf("%d", getInt(m, "processed_count")),
				fmt.Sprintf("%d", getInt(m, "found_count")),
				fmt.Sprintf("%d", getInt(m, "credits_used")),
			})
		}
	}
	if jt == "password" {
		output.Table([]string{"ID", "Status", "Total", "Processed", "Breached", "Credits"}, rows)
	} else {
		output.Table([]string{"ID", "Status", "Total", "Processed", "Found", "Credits"}, rows)
	}
	return nil
}

// ── status ──────────────────────────────────────────────────────────────────

func statusNonValidityJob(cmd *cobra.Command, args []string, jt string) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	spinner := startSpinner("Loading job...")
	data, err := fetchJobStatus(cmd, client, jt, args[0])
	stopSpinner(spinner)
	if err != nil {
		return jobsError(err)
	}
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	var m map[string]interface{}
	if !decode(data, &m) {
		return nil
	}
	output.Header(jobTypeTitle(jt) + " Job: " + args[0])
	if jt == "password" {
		printPasswordJob(m)
	} else {
		printIdentityJob(m)
	}
	return nil
}

// ── results ─────────────────────────────────────────────────────────────────

func resultsNonValidityJob(cmd *cobra.Command, args []string, jt string) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	breached, _ := cmd.Flags().GetBool("breached")
	foundOnly, _ := cmd.Flags().GetBool("found-only")

	spinner := startSpinner("Loading results...")
	data, err := fetchJobResults(cmd, client, jt, args[0], page, pageSize, breached, foundOnly)
	stopSpinner(spinner)
	if err != nil {
		return jobsError(err)
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
		if jt == "password" {
			rows = append(rows, []string{
				fmt.Sprintf("%d", getInt(m, "line_no")),
				firstNonEmpty(getStr(m, "prefix"), "—"),
				yesNo(boolField(m, "found")),
				fmt.Sprintf("%d", getInt(m, "count")),
			})
		} else {
			rows = append(rows, []string{
				firstNonEmpty(getStr(m, "email"), "—"),
				yesNo(boolField(m, "found")),
				firstNonEmpty(getStr(m, "name"), "—"),
				firstNonEmpty(getStr(m, "company"), "—"),
				firstNonEmpty(getStr(m, "job_role"), "—"),
				firstNonEmpty(getStr(m, "location"), "—"),
			})
		}
	}
	if jt == "password" {
		output.Table([]string{"Line", "Prefix", "Breached", "Count"}, rows)
	} else {
		output.Table([]string{"Email", "Found", "Name", "Company", "Role", "Location"}, rows)
	}
	return nil
}

// ── download ────────────────────────────────────────────────────────────────

func downloadNonValidityJob(cmd *cobra.Command, args []string, jt string) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	breached, _ := cmd.Flags().GetBool("breached")
	foundOnly, _ := cmd.Flags().GetBool("found-only")
	out, _ := cmd.Flags().GetString("out")

	spinner := startSpinner("Downloading results...")
	var blob []byte
	if jt == "password" {
		blob, err = client.DownloadPasswordJob(cmd.Context(), args[0], breached)
	} else {
		blob, err = client.DownloadIdentityJob(cmd.Context(), args[0], foundOnly || breached)
	}
	stopSpinner(spinner)
	if err != nil {
		return jobsError(err)
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
}

// ── cancel ──────────────────────────────────────────────────────────────────

func cancelNonValidityJob(cmd *cobra.Command, args []string, jt string) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	spinner := startSpinner("Cancelling job...")
	data, err := cancelJob(cmd, client, jt, args[0])
	stopSpinner(spinner)
	if err != nil {
		return jobsError(err)
	}
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	output.SuccessMsg("Job cancelled: " + args[0])
	return nil
}

// ── retry ───────────────────────────────────────────────────────────────────

func retryNonValidityJob(cmd *cobra.Command, args []string, jt string) error {
	if jt == "password" {
		return friendlyFormatError(cmd, "password jobs cannot be retried")
	}
	client, err := newClient()
	if err != nil {
		return err
	}
	spinner := startSpinner("Retrying job...")
	data, err := client.RetryIdentityJob(cmd.Context(), args[0])
	stopSpinner(spinner)
	if err != nil {
		return jobsError(err)
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
}

// jobsError prints an API error and returns it so the command exits non-zero.
func jobsError(err error) error {
	output.Error(err.Error())
	return err
}
