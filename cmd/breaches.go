package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var breachesCmd = &cobra.Command{
	Use:   "breaches",
	Short: "Scan sources for leaked secrets",
	Long: `Scan external sources for leaked credentials and secrets.

Examples:
  encrata breaches github scan https://github.com/org/repo
  encrata breaches github get <scan-id>`,
}

var breachesGithubCmd = &cobra.Command{
	Use:   "github",
	Short: "Scan a GitHub repository for leaked secrets",
}

var breachesGithubScanCmd = &cobra.Command{
	Use:   "scan [repo]",
	Short: "Start a secret scan of a GitHub repository",
	Long: `Scan a GitHub repository for committed secrets and credentials.

By default only the current checkout is scanned and the result comes back inline.
Use --deep to walk the full commit history; a deep scan runs in the background and
returns a scan ID whose result you fetch later with ` + "`encrata breaches github get`" + `.

Exit codes: by default the command exits 0 even when a secret is found. Pass
--fail-on-finding to exit 2 when any secret is detected (3 = auth, 4 = credits).

Examples:
  encrata breaches github scan https://github.com/org/repo
  encrata breaches github scan https://github.com/org/repo --deep
  encrata breaches github scan https://github.com/org/repo --ref main --min-severity high`,
	Args: cobra.ExactArgs(1),
	RunE: runGithubScan,
}

var breachesGithubGetCmd = &cobra.Command{
	Use:   "get [scan-id]",
	Short: "Fetch the result of a GitHub secret scan",
	Long: `Fetch a previously started GitHub secret scan by its ID.

Examples:
  encrata breaches github get scn_abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runGithubScanGet,
}

func init() {
	breachesGithubScanCmd.Flags().Bool("deep", false, "Scan the full commit history (runs in the background)")
	breachesGithubScanCmd.Flags().String("ref", "", "Branch, tag or commit to scan")
	breachesGithubScanCmd.Flags().String("min-severity", "", "Only report findings at or above this severity (low, medium, high, critical)")
	breachesGithubScanCmd.Flags().Bool("fail-on-finding", false, "Exit with code 2 if any secret is found")
	breachesGithubScanCmd.Flags().String("out", "", "Write the full JSON result to a file")

	breachesGithubGetCmd.Flags().Bool("fail-on-finding", false, "Exit with code 2 if any secret is found")
	breachesGithubGetCmd.Flags().String("out", "", "Write the full JSON result to a file")

	breachesGithubCmd.AddCommand(breachesGithubScanCmd, breachesGithubGetCmd)
	breachesCmd.AddCommand(breachesGithubCmd)
}

func runGithubScan(cmd *cobra.Command, args []string) error {
	repo := strings.TrimSpace(args[0])
	if repo == "" {
		return friendlyFormatError(cmd, "provide the GitHub repository URL to scan")
	}

	deep, _ := cmd.Flags().GetBool("deep")
	ref, _ := cmd.Flags().GetString("ref")
	minSeverity, _ := cmd.Flags().GetString("min-severity")

	req := api.GitHubScanRequest{
		Repo: repo,
		Config: api.GitHubScanConfig{
			DeepScan:    deep,
			Ref:         strings.TrimSpace(ref),
			MinSeverity: strings.TrimSpace(minSeverity),
		},
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	spinner := startSpinner("Scanning repository...")
	data, err := client.ScanGitHubRepo(cmd.Context(), req)
	stopSpinner(spinner)
	if err != nil {
		output.Error(err.Error())
		return err
	}
	return renderScanResult(cmd, data, "GitHub Secret Scan: "+repo)
}

func runGithubScanGet(cmd *cobra.Command, args []string) error {
	id := strings.TrimSpace(args[0])
	if id == "" {
		return friendlyFormatError(cmd, "provide the scan ID to fetch")
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	spinner := startSpinner("Fetching scan result...")
	data, err := client.GetGitHubScan(cmd.Context(), id)
	stopSpinner(spinner)
	if err != nil {
		output.Error(err.Error())
		return err
	}
	return renderScanResult(cmd, data, "GitHub Secret Scan: "+id)
}

// renderScanResult prints an enveloped scan result and, when --fail-on-finding
// is set, returns errBreachDetected so Execute() exits with the findings code.
func renderScanResult(cmd *cobra.Command, data json.RawMessage, title string) error {
	out, _ := cmd.Flags().GetString("out")

	var result map[string]interface{}
	if jsonMode() {
		output.JSON(data)
		_ = json.Unmarshal(data, &result)
	} else {
		if !decode(data, &result) {
			return nil
		}
		output.Header(title)
		renderGitHubScan(result)
		if msg := field(result, "message"); msg != "" {
			output.Dim.Println("  " + msg)
		}
		output.Dim.Printf("  Credits used: %s\n", firstNonEmpty(field(result, "result.credits", "credits"), "0"))
	}

	if out != "" {
		if err := saveResult(out, data); err != nil {
			return err
		}
	}

	total := intOf(countField(result, "result.summary.total", "result.findings"))
	if total > 0 && failOnFinding(cmd) {
		return errBreachDetected
	}
	return nil
}

func renderGitHubScan(r map[string]interface{}) {
	printNonEmptyKV(
		"Scan ID", field(r, "result.scan_id"),
		"Repository", field(r, "result.target"),
		"Commit", field(r, "result.commit_sha"),
		"Status", firstNonEmpty(field(r, "result.status"), "succeeded"),
	)

	total := intOf(countField(r, "result.summary.total", "result.findings"))
	label := output.Success.Sprint("0")
	if total > 0 {
		label = output.Err.Sprint(fmt.Sprintf("%d", total))
	}
	output.KV("Secrets found", label)
	fmt.Println()

	renderFindingsTable(r)
}

// renderFindingsTable prints a Severity | Rule | File | Line table when findings exist.
func renderFindingsTable(r map[string]interface{}) {
	arr := firstArr(r, "result.findings")
	if len(arr) == 0 {
		return
	}
	rows := make([][]string, 0, len(arr))
	for _, f := range arr {
		m := asMap(f)
		rows = append(rows, []string{
			firstNonEmpty(field(m, "severity"), "-"),
			firstNonEmpty(field(m, "rule_id", "description"), "-"),
			firstNonEmpty(field(m, "file"), "-"),
			firstNonEmpty(field(m, "start_line"), "-"),
		})
	}
	output.Table([]string{"Severity", "Rule", "File", "Line"}, rows)
	fmt.Println()
}
