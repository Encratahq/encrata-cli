package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const version = "0.3.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("  %s %s\n", "\033[1;38;5;173mencrata\033[0m", "\033[38;5;245mv"+version+"\033[0m")
	},
}
