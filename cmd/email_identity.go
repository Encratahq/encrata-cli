package cmd

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var emailIdentityCmd = &cobra.Command{
	Use:   "identity [email|file]",
	Short: "Person identity: name, role, company, socials, breaches (1 credit)",
	Long: `Look up the person, work history, education, and social profiles associated
with an email address.

Pass a single email to resolve it, or use --bulk with a file (or - for STDIN)
to resolve a whole list concurrently.

Examples:
  encrata email identity user@example.com
  encrata email identity emails.csv --bulk
  encrata email identity emails.csv --bulk --out people.csv`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bulk, _ := cmd.Flags().GetBool("bulk")
		if bulk {
			path := ""
			if len(args) == 1 {
				path = args[0]
			}
			return runIdentityBulk(cmd, path)
		}
		if len(args) != 1 {
			return friendlyFormatError(cmd, "provide one email address, or use --bulk with a file")
		}
		full, _ := cmd.Flags().GetBool("full")
		return emailLookup(cmd, args[0], "Identity", "Resolving identity...",
			api.API.EmailIdentity,
			renderIdentity(full),
			nil)
	},
}

func init() {
	emailIdentityCmd.Flags().Bool("full", false, "Show the full breach list")
	emailIdentityCmd.Flags().Bool("bulk", false, "Resolve a file (or - for STDIN) of emails concurrently")
	emailIdentityCmd.Flags().Int("concurrency", 8, "Parallel lookups in --bulk mode")
	emailIdentityCmd.Flags().String("format", "", "Bulk export format: csv, xlsx, or json (default: inferred from --out)")
	emailIdentityCmd.Flags().String("only", "", "Export only rows matching: found")
	emailIdentityCmd.Flags().Bool("found-only", false, "Deprecated: use --only found")
	_ = emailIdentityCmd.Flags().MarkHidden("found-only")
}

// runIdentityBulk resolves identities for a file/STDIN list concurrently,
// prints a summary count, and optionally exports the rows.
func runIdentityBulk(cmd *cobra.Command, path string) error {
	if err := validateOnly(cmd, "found"); err != nil {
		return err
	}
	_, emails, _, err := loadEmails(cmd, path)
	if err != nil {
		return err
	}
	client, err := newClient()
	if err != nil {
		return err
	}
	out, _ := cmd.Flags().GetString("out")

	concurrency, _ := cmd.Flags().GetInt("concurrency")
	if concurrency < 1 {
		concurrency = 1
	}
	total := len(emails)
	if concurrency > total && total > 0 {
		concurrency = total
	}

	asJSON := jsonMode()
	if !asJSON {
		output.Header(fmt.Sprintf("Bulk Identity: %d %s", total, plural(total, "email", "emails")))
	}

	results := make([]map[string]interface{}, total)
	rawResults := make([]json.RawMessage, total)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var done int32
	var mu sync.Mutex

	for i, email := range emails {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, addr string) {
			defer wg.Done()
			defer func() { <-sem }()

			data, err := client.EmailIdentity(cmd.Context(), addr)
			row := map[string]interface{}{"email": addr}
			if err != nil {
				row["error"] = err.Error()
			} else if m, ok := decodeRow(data); ok {
				row = m
				if _, hasEmail := row["email"]; !hasEmail {
					row["email"] = addr
				}
				copyRaw := make(json.RawMessage, len(data))
				copy(copyRaw, data)
				rawResults[idx] = copyRaw
			}
			results[idx] = row

			if !asJSON {
				mu.Lock()
				done++
				renderProgress(int(done), total)
				mu.Unlock()
			}
		}(i, email)
	}
	wg.Wait()

	cleanRaw := make([]json.RawMessage, 0, total)
	for _, raw := range rawResults {
		if len(raw) > 0 {
			cleanRaw = append(cleanRaw, raw)
		}
	}

	found := countFoundIdentities(results)

	if asJSON {
		output.JSON(rawResultsArray(cleanRaw))
		if out != "" {
			return exportIdentity(cmd, out, results)
		}
		return nil
	}

	fmt.Println()
	fmt.Println()
	fmt.Printf("  Checked %d · Found %s · Not found %s · Credits %d\n",
		total,
		output.Success.Sprintf("%d", found),
		output.Dim.Sprintf("%d", total-found),
		sumCredits(results),
	)
	// Bulk always persists rows; exportIdentity auto-names the file when --out is empty.
	return exportIdentity(cmd, out, results)
}
