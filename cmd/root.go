package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/config"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// Exit codes. Findings (breaches) are distinct from operational failures so
// callers can tell a successful scan from a broken one.
const (
	exitOK      = 0
	exitError   = 1 // generic/operational error
	exitFinding = 2 // a breach/finding was detected (opt-in via --fail-on-finding)
	exitAuth    = 3 // authentication failed (401)
	exitCredits = 4 // insufficient credits (402)
)

var (
	cfg        *config.Config
	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "encrata",
	Short: "Encrata CLI — intelligence lookups from your terminal",
	Long: fmt.Sprintf(`
  %s
  %s

  Get started:
    %s
    %s
    %s

  Docs: %s`,
		"\033[1;38;5;173mencrata\033[0m",
		"\033[38;5;245mintelligence lookups from your terminal\033[0m",
		"\033[38;5;173mencrata config set-key\033[0m <your-api-key>",
		"\033[38;5;173mencrata email validity\033[0m user@example.com",
		"\033[38;5;173mencrata email bulk\033[0m emails.csv --out results.csv",
		"\033[38;5;109mhttps://docs.encrata.com\033[0m"),
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() int {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := rootCmd.ExecuteContext(ctx)
	if err == nil {
		return exitOK
	}
	if ctx.Err() != nil {
		output.Error("Aborted.")
		return exitError
	}
	// A detected breach is a successful check with a distinct exit code; the
	// verdict is already rendered, so don't print an error banner.
	if errors.Is(err, errBreachDetected) {
		return exitFinding
	}
	// Map auth/credit failures to dedicated codes so scripts can react.
	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusUnauthorized:
			output.Error(err.Error())
			return exitAuth
		case http.StatusPaymentRequired:
			output.Error(err.Error())
			return exitCredits
		}
	}
	output.Error(err.Error())
	return exitError
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().String("api-key", "", "API key (overrides config/env)")
	rootCmd.PersistentFlags().String("base-url", "", "API base URL (overrides config/env)")
	rootCmd.PersistentFlags().Bool("quiet", false, "Suppress decorative output (headers, spinner)")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().Int("timeout", 0, "Per-request timeout in seconds (0 = default 90s)")

	rootCmd.AddCommand(emailCmd)
	rootCmd.AddCommand(passwordCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(keysCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(jobsCmd)
	rootCmd.AddCommand(identityJobsCmd)
	rootCmd.AddCommand(passwordJobsCmd)
	rootCmd.AddCommand(listsCmd)
	rootCmd.AddCommand(webhooksCmd)
	rootCmd.AddCommand(workspaceCmd)
	improveArgErrors(rootCmd)
}

func initConfig() {
	cfg = config.Load()

	api.Version = version

	if key, _ := rootCmd.PersistentFlags().GetString("api-key"); key != "" {
		cfg.APIKey = key
	}
	if url, _ := rootCmd.PersistentFlags().GetString("base-url"); url != "" {
		cfg.BaseURL = url
	}
	if jsonOutput {
		cfg.Output = "json"
	}
	if quiet, _ := rootCmd.PersistentFlags().GetBool("quiet"); quiet {
		output.SetQuiet(true)
	}
	if noColor, _ := rootCmd.PersistentFlags().GetBool("no-color"); noColor {
		output.SetColor(false)
	}
	if secs, _ := rootCmd.PersistentFlags().GetInt("timeout"); secs > 0 {
		api.RequestTimeout = time.Duration(secs) * time.Second
	}
}
