package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/validation"
	"github.com/spf13/cobra"
)

var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Email intelligence lookups",
	Long: `Validate, enrich, and investigate email addresses.

Examples:
  encrata email validity  user@example.com
  encrata email enrich    user@example.com
  encrata email identity  user@example.com
  encrata email breaches  user@example.com
  encrata email verify    user@example.com
  encrata email bulk      emails.csv --out results.csv`,
}

func init() {
	emailCmd.AddCommand(
		emailValidityCmd,
		emailEnrichCmd,
		emailIdentityCmd,
		emailBreachesCmd,
		emailVerifyCmd,
		emailBulkCmd,
	)

	// Single-email commands can write the full JSON payload to a file.
	for _, c := range []*cobra.Command{
		emailValidityCmd,
		emailEnrichCmd,
		emailIdentityCmd,
		emailBreachesCmd,
		emailVerifyCmd,
	} {
		c.Flags().String("out", "", "Write the full JSON result to a file")
	}
}

// emailLookup runs a single-email API call and renders the result, mirroring the
// spinner / --json / header conventions used across the CLI. footer prints the
// trailing line (credits or a free note); when nil it defaults to printCredits.
func emailLookup(
	cmd *cobra.Command,
	email, title, spinnerMsg string,
	call func(api.API, context.Context, string) (json.RawMessage, error),
	render func(map[string]interface{}),
	footer func(map[string]interface{}),
) error {
	if err := validation.Email(email); err != nil {
		return friendlyFormatError(cmd, err.Error())
	}
	client, err := newClient()
	if err != nil {
		return err
	}

	spinner := startSpinner(spinnerMsg)
	data, err := call(client, cmd.Context(), email)
	stopSpinner(spinner)
	if err != nil {
		output.Error(err.Error())
		return err
	}

	out, _ := cmd.Flags().GetString("out")

	if jsonMode() {
		output.JSON(data)
		if out != "" {
			return saveResult(out, data)
		}
		return nil
	}

	var result map[string]interface{}
	if !decode(data, &result) {
		return nil
	}

	output.Header(title + ": " + email)
	render(result)
	if footer != nil {
		footer(result)
	} else {
		printCredits(result)
	}
	if out != "" {
		return saveResult(out, data)
	}
	return nil
}

// printCredits prints the credit cost of a response, defaulting to 0.
func printCredits(result map[string]interface{}) {
	output.Dim.Printf("  Credits used: %s\n", creditsValue(result))
}

// freeFooter notes that a command does not consume credits.
func freeFooter(map[string]interface{}) {
	output.Dim.Println("  Free — no credits used")
}

// loadEmails reads emails from a file path (or STDIN when path is empty or "-"),
// parses CSV/line content, validates and de-duplicates them, and returns the
// unique emails alongside the raw bytes and a display file name.
func loadEmails(cmd *cobra.Command, path string) (fileName string, emails []string, raw []byte, err error) {
	fileName = "list.csv"
	if path == "" || path == "-" {
		raw, err = io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to read from stdin: %w", err)
		}
	} else {
		raw, err = os.ReadFile(path)
		if err != nil {
			return "", nil, nil, err
		}
		fileName = filepath.Base(path)
	}

	emails = parseEmails(raw)
	if len(emails) == 0 {
		return "", nil, nil, fmt.Errorf("no valid email addresses found in input")
	}
	return fileName, emails, raw, nil
}

// parseEmails extracts unique valid email addresses from CSV/line input.
func parseEmails(raw []byte) []string {
	seen := make(map[string]bool)
	var emails []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" {
			continue
		}
		for _, cell := range strings.Split(line, ",") {
			cell = strings.TrimSpace(cell)
			key := strings.ToLower(cell)
			if validation.Email(cell) == nil && !seen[key] {
				seen[key] = true
				emails = append(emails, cell)
			}
		}
	}
	return emails
}

// renderProgress draws an in-place progress bar for streaming/polling work.
func renderProgress(done, total int) {
	const width = 30
	ratio := 0.0
	if total > 0 {
		ratio = float64(done) / float64(total)
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Printf("\r  %s %d/%d", output.Brand.Sprint(bar), done, total)
}

// writeResults, writeCSV and cellString were replaced by the flattened bulk
// exporter in email_export.go.
