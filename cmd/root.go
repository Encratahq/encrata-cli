package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/config"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
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
		"\033[38;5;173mencrata email\033[0m user@example.com",
		"\033[38;5;173mencrata ip\033[0m 8.8.8.8",
		"\033[38;5;109mhttps://docs.encrata.com\033[0m"),
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		if ctx.Err() != nil {
			output.Error("Aborted.")
			return nil
		}
		output.Error(err.Error())
	}
	return err
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().String("api-key", "", "API key (overrides config/env)")
	rootCmd.PersistentFlags().String("base-url", "", "API base URL (overrides config/env)")

	rootCmd.AddCommand(emailCmd)
	rootCmd.AddCommand(phoneCmd)
	rootCmd.AddCommand(companyCmd)
	rootCmd.AddCommand(domainCmd)
	rootCmd.AddCommand(ipCmd)
	rootCmd.AddCommand(googleCmd)
	rootCmd.AddCommand(darkwebCmd)
	rootCmd.AddCommand(scrapeCmd)
	rootCmd.AddCommand(extractCmd)
	rootCmd.AddCommand(screenshotCmd)
	rootCmd.AddCommand(faceCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(breachesCmd)
	rootCmd.AddCommand(bulkCmd)
	rootCmd.AddCommand(listsCmd)
	rootCmd.AddCommand(monitorsCmd)
	rootCmd.AddCommand(workflowsCmd)
	rootCmd.AddCommand(keysCmd)
	rootCmd.AddCommand(webhooksCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(jobsCmd)
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
}
