package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var darkwebCmd = &cobra.Command{
	Use:   "darkweb [query]",
	Short: "Search the dark web for leaked data",
	Long:  "Search dark web sources for breaches, leaks, and mentions related to a query (email, domain, keyword).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.New(cfg.BaseURL, cfg.APIKey)

		offset, _ := cmd.Flags().GetInt("offset")

		req := &api.DarkwebRequest{
			Query:  args[0],
			Offset: offset,
		}

		data, err := client.DarkwebSearch(cmd.Context(), req)
		if err != nil {
			output.Error(err.Error())
			return err
		}

		if cfg.Output == "json" {
			output.JSON(data)
			return nil
		}

		var result struct {
			Query       string `json:"query"`
			Total       int    `json:"total"`
			ResultCount int    `json:"result_count"`
			Results     []struct {
				Source  string `json:"source"`
				Domain  string `json:"domain"`
				Title   string `json:"title"`
				Snippet string `json:"snippet"`
				Date    string `json:"date"`
			} `json:"results"`
			Credits int `json:"credits"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		output.Header("Dark Web Search: " + args[0])
		output.Dim.Printf("  %d results found (showing %d)\n\n", result.Total, result.ResultCount)

		for i, r := range result.Results {
			output.Bold.Printf("  [%d] %s\n", i+1, r.Title)
			if r.Source != "" {
				output.KV("Source", r.Source)
			}
			if r.Domain != "" {
				output.KV("Domain", r.Domain)
			}
			if r.Date != "" {
				output.KV("Date", r.Date)
			}
			if r.Snippet != "" {
				output.Dim.Printf("    %s\n", r.Snippet)
			}
			fmt.Println()
		}

		output.Dim.Printf("  Credits used: %d\n", result.Credits)
		return nil
	},
}

func init() {
	darkwebCmd.Flags().Int("offset", 0, "Pagination offset")
}
