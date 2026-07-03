package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Show how to update the CLI",
	Run: func(cmd *cobra.Command, args []string) {
		output.Header("Update Encrata CLI")
		fmt.Printf("  Current version: v%s\n\n", version)
		fmt.Println("  If installed with npm:")
		fmt.Println("    npm install -g encrata-cli@latest")
		fmt.Println()
		fmt.Println("  If installed with Homebrew:")
		fmt.Println("    brew upgrade encrata")
		fmt.Println()
		fmt.Println("  If installed from source:")
		fmt.Println("    go install github.com/Encratahq/cli@latest")
		fmt.Println()
	},
}
