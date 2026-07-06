package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

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
				Source    string   `json:"source"`
				Network   string   `json:"network"`
				Domain    string   `json:"domain"`
				Title     string   `json:"title"`
				URL       string   `json:"url"`
				Snippet   string   `json:"snippet"`
				Body      string   `json:"body"`
				PostDate  string   `json:"post_date"`
				CrawlDate string   `json:"crawl_date"`
				Emails    []string `json:"emails"`
				Password  string   `json:"password"`
				HashType  string   `json:"hash_type"`
				Context   string   `json:"context"`
				Leak      *struct {
					Name string `json:"name"`
				} `json:"leak"`
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
			if r.Network != "" {
				output.KV("Network", r.Network)
			}
			if r.Domain != "" {
				output.KV("Domain", r.Domain)
			}
			if r.URL != "" {
				output.KV("URL", r.URL)
			}
			if r.Leak != nil && r.Leak.Name != "" {
				output.KV("Leak", r.Leak.Name)
			}
			if r.PostDate != "" {
				output.KV("Post Date", r.PostDate)
			}
			if r.CrawlDate != "" {
				output.KV("Crawl Date", r.CrawlDate)
			}
			if len(r.Emails) > 0 {
				output.KV("Emails", strings.Join(r.Emails, ", "))
			}
			if r.Password != "" {
				output.KV("Password", r.Password)
			}
			if r.HashType != "" {
				output.KV("Hash Type", r.HashType)
			}
			if r.Context != "" {
				output.KV("Context", truncate(r.Context, 140))
			}
			text := firstNonEmpty(r.Snippet, r.Body)
			if text != "" {
				output.Dim.Printf("    %s\n", truncate(text, 180))
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

var darkwebCrawlCmd = &cobra.Command{
	Use:   "crawl [onion-url]",
	Short: "Crawl a .onion site over Tor",
	Long:  "Crawl a .onion URL over Tor and extract page titles, emails, phone numbers, and linked onion services.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		depth, _ := cmd.Flags().GetInt("depth")
		force, _ := cmd.Flags().GetBool("force")

		req := &api.DarkwebCrawlRequest{
			URL:   args[0],
			Depth: depth,
			Force: force,
		}

		data, err := client.DarkwebCrawl(cmd.Context(), req)
		if err != nil {
			return err
		}

		if jsonMode() {
			output.JSON(data)
			return nil
		}

		var result struct {
			URL     string   `json:"url"`
			Depth   int      `json:"depth"`
			Live    bool     `json:"live"`
			Count   int      `json:"count"`
			Emails  []string `json:"emails"`
			Onions  []string `json:"onions"`
			Credits float64  `json:"credits"`
			Nodes   []struct {
				URL    string   `json:"url"`
				Title  string   `json:"title"`
				Status int      `json:"status"`
				Live   bool     `json:"live"`
				Emails []string `json:"emails"`
				Phones []string `json:"phones"`
				Links  []string `json:"links"`
			} `json:"nodes"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		output.Header("Dark Web Crawl: " + result.URL)
		output.KV(
			"Depth", fmt.Sprintf("%d", result.Depth),
			"Live", fmt.Sprintf("%t", result.Live),
			"Pages", fmt.Sprintf("%d", result.Count),
			"Emails", fmt.Sprintf("%d", len(result.Emails)),
			"Onions", fmt.Sprintf("%d", len(result.Onions)),
		)
		fmt.Println()

		for i, node := range result.Nodes {
			if i >= 10 {
				break
			}
			title := firstNonEmpty(node.Title, node.URL)
			output.Bold.Printf("  [%d] %s\n", i+1, title)
			output.KV(
				"URL", node.URL,
				"Status", fmt.Sprintf("%d", node.Status),
				"Live", fmt.Sprintf("%t", node.Live),
			)
			if len(node.Emails) > 0 {
				output.KV("Emails", strings.Join(node.Emails, ", "))
			}
			if len(node.Phones) > 0 {
				output.KV("Phones", strings.Join(node.Phones, ", "))
			}
			if len(node.Links) > 0 {
				output.KV("Links", fmt.Sprintf("%d", len(node.Links)))
			}
			fmt.Println()
		}

		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}

func init() {
	darkwebCmd.Flags().Int("offset", 0, "Pagination offset")
	darkwebCrawlCmd.Flags().Int("depth", 1, "Crawl depth from 1 to 3")
	darkwebCrawlCmd.Flags().Bool("force", false, "Bypass cache and run a fresh billed crawl")
	darkwebCmd.AddCommand(darkwebCrawlCmd)
}
