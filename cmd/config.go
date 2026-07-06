package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Encratahq/cli/internal/config"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

var setKeyCmd = &cobra.Command{
	Use:   "set-key [key]",
	Short: "Set your Encrata API key",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var key string
		if len(args) > 0 {
			key = args[0]
		} else {
			fmt.Print("  Enter API key: ")
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read API key: %w", err)
			}
			key = strings.TrimSpace(input)
		}

		if key == "" {
			return fmt.Errorf("API key cannot be empty")
		}

		cfg.APIKey = key
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		output.Success.Println("  ✓ API key saved to ~/.encrata/config.yaml")
		return nil
	},
}

var setURLCmd = &cobra.Command{
	Use:   "set-url [url]",
	Short: "Set custom API base URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.BaseURL = args[0]
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		output.Success.Printf("  ✓ Base URL set to %s\n", args[0])
		return nil
	},
}

var showConfigCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		output.Header("Configuration")
		maskedKey := "not set"
		if cfg.APIKey != "" {
			if len(cfg.APIKey) > 8 {
				maskedKey = cfg.APIKey[:4] + "..." + cfg.APIKey[len(cfg.APIKey)-4:]
			} else {
				maskedKey = "****"
			}
		}
		output.KV(
			"API Key", maskedKey,
			"Base URL", cfg.BaseURL,
			"Output", cfg.Output,
		)
		fmt.Println()
	},
}

func init() {
	configCmd.AddCommand(setKeyCmd)
	configCmd.AddCommand(setURLCmd)
	configCmd.AddCommand(showConfigCmd)
}
