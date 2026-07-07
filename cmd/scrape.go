package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/validation"
	"github.com/spf13/cobra"
)

var scrapeCmd = &cobra.Command{
	Use:   "scrape [url]",
	Short: "Scrape a web page's raw HTML",
	Long:  "Fetch the raw HTML of a web page. Renders JavaScript, bypasses bot blocks, and blocks ads/trackers by default.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.New(cfg.BaseURL, cfg.APIKey)

		req := &api.ScrapeRequest{URL: args[0]}
		if cmd.Flags().Changed("no-js") {
			v := false
			req.RenderJS = &v
		}
		if waitFor, _ := cmd.Flags().GetString("wait-for"); waitFor != "" {
			req.WaitFor = waitFor
		}
		if timeout, _ := cmd.Flags().GetInt("timeout"); cmd.Flags().Changed("timeout") {
			if err := validation.Timeout(timeout); err != nil {
				return err
			}
			if timeout > 0 {
				req.Timeout = timeout
			}
		}

		spinner := startSpinner("Scraping page...")
		data, err := client.Scrape(cmd.Context(), req)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}

		if cfg.Output == "json" {
			output.JSON(data)
			return nil
		}

		var result struct {
			Success    bool   `json:"success"`
			URL        string `json:"url"`
			StatusCode int    `json:"status_code"`
			Content    string `json:"content"`
			Error      string `json:"error"`
			ErrorCode  string `json:"error_code"`
			Metadata   *struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			} `json:"metadata"`
			Credits float64 `json:"credits"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		if !result.Success {
			msg := result.Error
			if msg == "" {
				msg = result.ErrorCode
			}
			output.Error("scrape failed: " + msg)
			return fmt.Errorf("scrape failed")
		}

		outFile, _ := cmd.Flags().GetString("output-file")
		if outFile != "" {
			if err := os.WriteFile(outFile, []byte(result.Content), 0o644); err != nil {
				output.Error(err.Error())
				return err
			}
			output.Header("Scrape: " + result.URL)
			output.KV("Status", fmt.Sprintf("%d", result.StatusCode), "Saved to", outFile, "Size", fmt.Sprintf("%d bytes", len(result.Content)))
			output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
			return nil
		}

		output.Header("Scrape: " + result.URL)
		if result.Metadata != nil && result.Metadata.Title != "" {
			output.KV("Title", result.Metadata.Title)
		}
		output.KV("Status", fmt.Sprintf("%d", result.StatusCode), "Size", fmt.Sprintf("%d bytes", len(result.Content)))
		fmt.Println()
		fmt.Println(result.Content)
		fmt.Println()
		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}

func init() {
	scrapeCmd.Flags().Bool("no-js", false, "Disable JavaScript rendering")
	scrapeCmd.Flags().String("wait-for", "", "CSS selector to wait for before capturing")
	scrapeCmd.Flags().Int("timeout", 0, "Timeout in milliseconds (max 60000)")
	scrapeCmd.Flags().StringP("output-file", "o", "", "Write HTML to a file instead of stdout")
}
