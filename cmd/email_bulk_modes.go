package cmd

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// runBulkEnrich runs the FULL single-email validity report for every address
// concurrently, so exported rows carry every column (confidence, provider, mx,
// domain_trust, smtp, footprint, ...) instead of just status/reason. Costs 1
// credit per email.
func runBulkEnrich(cmd *cobra.Command, client api.API, emails []string, out string, fields []string) error {
	total := len(emails)
	asJSON := jsonMode()
	if !asJSON {
		output.Header(fmt.Sprintf("Bulk Enrich: %d %s", total, plural(total, "email", "emails")))
	}

	concurrency, _ := cmd.Flags().GetInt("concurrency")
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > total && total > 0 {
		concurrency = total
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

			data, err := client.EmailValidity(cmd.Context(), addr)
			row := map[string]interface{}{"email": addr}
			if err != nil {
				row["status"] = "error"
				row["reason"] = err.Error()
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

	// Drop rows that failed to decode into a payload (nil raw) for the JSON array.
	cleanRaw := make([]json.RawMessage, 0, total)
	for _, raw := range rawResults {
		if len(raw) > 0 {
			cleanRaw = append(cleanRaw, raw)
		}
	}

	if asJSON {
		output.JSON(rawResultsArray(cleanRaw))
		if out != "" {
			return exportBulk(cmd, out, results)
		}
		return nil
	}

	fmt.Println()
	fmt.Println()
	printBulkSummaryLine(results)
	// Bulk always persists rows; exportBulk auto-names the file when --out is empty.
	return exportBulk(cmd, out, results)
}

func runBulkStream(cmd *cobra.Command, client api.API, emails []string, fileName, out string, fields []string) error {
	total := len(emails)
	asJSON := jsonMode()
	if !asJSON {
		output.Header(fmt.Sprintf("Bulk Validity: %d %s", total, plural(total, "email", "emails")))
	}

	var results []map[string]interface{}
	var rawResults []json.RawMessage
	done := 0
	streamErr := client.BulkValiditySearch(cmd.Context(), emails, fileName, func(ev api.BulkEvent) error {
		switch ev.Type {
		case "result":
			var m map[string]interface{}
			if json.Unmarshal(ev.Data, &m) == nil {
				results = append(results, m)
				raw := make(json.RawMessage, len(ev.Data))
				copy(raw, ev.Data)
				rawResults = append(rawResults, raw)
			}
			done++
			if !asJSON {
				renderProgress(done, total)
			}
		case "error":
			if !asJSON {
				var e map[string]interface{}
				if json.Unmarshal(ev.Data, &e) == nil {
					output.Error(getStr(e, "error"))
				}
			}
		}
		return nil
	})
	if !asJSON {
		fmt.Println()
	}
	if streamErr != nil {
		output.Error(streamErr.Error())
		return streamErr
	}

	if asJSON {
		output.JSON(rawResultsArray(rawResults))
		if out != "" {
			return exportBulk(cmd, out, results)
		}
		return nil
	}

	fmt.Println()
	printBulkSummaryLine(results)
	// Bulk always persists rows; exportBulk auto-names the file when --out is empty.
	return exportBulk(cmd, out, results)
}
