package cmd

import (
	"context"
	"errors"
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
		"\033[38;5;173mencrata email validity\033[0m user@example.com",
		"\033[38;5;173mencrata email bulk\033[0m emails.csv --out results.csv",
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
		// A detected breach is a successful check with a non-zero exit code;
		// the verdict is already rendered, so don't print an error banner.
		if errors.Is(err, errBreachDetected) {
			return err
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
	rootCmd.AddCommand(passwordCmd)
	rootCmd.AddCommand(keysCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(jobsCmd)
	rootCmd.AddCommand(listsCmd)

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
}
