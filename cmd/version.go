package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version, commit and build date",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("  %s %s\n", "\033[1;38;5;173mencrata\033[0m", "\033[38;5;245mv"+version.Version+"\033[0m")
		fmt.Printf("  %s\n", "\033[38;5;245mcommit "+version.Commit+" · built "+version.Date+"\033[0m")
	},
}
