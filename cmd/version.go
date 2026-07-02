package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is the CLI version. It defaults to the value below for local builds
// (go run / go build) and is overridden at release time by GoReleaser via
// -ldflags "-X github.com/Encratahq/cli/cmd.version=<git tag>".
var version = "0.4.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("  %s %s\n", "\033[1;38;5;173mencrata\033[0m", "\033[38;5;245mv"+version+"\033[0m")
	},
}
