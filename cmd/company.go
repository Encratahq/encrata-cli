package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/textutil"
	"github.com/spf13/cobra"
)

var companyCmd = &cobra.Command{
	Use:   "company [name or domain]",
	Short: "Look up a company",
	Long:  "Retrieve company details including industry, size, headquarters, and more.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.New(cfg.BaseURL, cfg.APIKey)
		data, err := client.CompanySearch(cmd.Context(), args[0])
		if err != nil {
			output.Error(err.Error())
			return err
		}

		if cfg.Output == "json" {
			output.JSON(data)
			return nil
		}

		var result struct {
			Query   string `json:"query"`
			Profile *struct {
				Name          string   `json:"name"`
				LegalName     string   `json:"legal_name"`
				Domain        string   `json:"domain"`
				Website       string   `json:"website"`
				Description   string   `json:"description"`
				Industry      string   `json:"industry"`
				Headquarters  string   `json:"headquarters"`
				Address       string   `json:"address"`
				City          string   `json:"city"`
				State         string   `json:"state"`
				Country       string   `json:"country"`
				CompanyType   string   `json:"company_type"`
				Founded       string   `json:"founded"`
				FoundingDate  string   `json:"founding_date"`
				EmployeeRange string   `json:"employee_range"`
				EmployeeCount int      `json:"employee_count"`
				Ticker        string   `json:"ticker"`
				StockSymbol   string   `json:"stock_symbol"`
				Sources       []string `json:"sources"`
			} `json:"profile"`
			Company *struct {
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Industry    string   `json:"industry"`
				Headquarter string   `json:"headquarter"`
				Website     string   `json:"website"`
				Employees   int      `json:"employees"`
				Followers   int      `json:"followers"`
				Specialties []string `json:"specialties"`
			} `json:"company"`
			KnowledgeGraph *struct {
				Title       string            `json:"title"`
				Type        string            `json:"type"`
				Description string            `json:"description"`
				Website     string            `json:"website"`
				Attributes  map[string]string `json:"attributes"`
			} `json:"knowledge_graph"`
			Organic []struct {
				Title    string `json:"title"`
				Link     string `json:"link"`
				Snippet  string `json:"snippet"`
				Position int    `json:"position"`
			} `json:"organic"`
			SECFilings []struct {
				CompanyName string `json:"companyName"`
				FormType    string `json:"formType"`
				FiledAt     string `json:"filedAt"`
				Description string `json:"description"`
			} `json:"sec_filings"`
			Credits float64 `json:"credits"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		output.Header("Company Lookup: " + args[0])
		shown := false

		if result.Profile != nil {
			shown = true
			p := result.Profile
			hq := firstNonEmpty(p.Headquarters, p.Address, joinNonEmpty(", ", p.City, p.State, p.Country))
			founded := firstNonEmpty(p.Founded, p.FoundingDate)
			employees := p.EmployeeRange
			if employees == "" && p.EmployeeCount > 0 {
				employees = fmt.Sprintf("%d", p.EmployeeCount)
			}

			output.Bold.Println("  Profile:")
			output.KV(
				"Name", firstNonEmpty(p.Name, p.LegalName),
				"Industry", p.Industry,
				"Website", firstNonEmpty(p.Website, p.Domain),
				"Ticker", firstNonEmpty(p.Ticker, p.StockSymbol),
				"Employees", employees,
				"HQ", hq,
				"Founded", founded,
				"Type", p.CompanyType,
				"Description", truncate(p.Description, 140),
			)
			if len(p.Sources) > 0 {
				fmt.Println()
				output.Bold.Println("  Sources:")
				for _, s := range p.Sources {
					fmt.Printf("    - %s\n", s)
				}
			}
			fmt.Println()
		}

		if result.Company != nil && result.Profile == nil {
			shown = true
			c := result.Company
			employees := ""
			if c.Employees > 0 {
				employees = fmt.Sprintf("%d", c.Employees)
			}
			output.Bold.Println("  Profile:")
			output.KV(
				"Name", c.Name,
				"Industry", c.Industry,
				"HQ", c.Headquarter,
				"Website", c.Website,
				"Employees", employees,
				"Description", truncate(c.Description, 140),
			)
			if len(c.Specialties) > 0 {
				fmt.Println()
				output.Bold.Println("  Specialties:")
				for _, s := range c.Specialties {
					fmt.Printf("    - %s\n", s)
				}
			}
			fmt.Println()
		}

		if result.KnowledgeGraph != nil {
			shown = true
			kg := result.KnowledgeGraph
			output.Bold.Println("  Knowledge Graph:")
			output.KV(
				"Type", kg.Type,
				"Founded", kg.Attributes["Founded"],
				"CEO", kg.Attributes["CEO"],
				"HQ", kg.Attributes["Headquarters"],
				"Website", kg.Website,
				"Description", truncate(kg.Description, 140),
			)
			fmt.Println()
		}

		if len(result.Organic) > 0 {
			shown = true
			output.Bold.Println("  Top Results:")
			for i, item := range result.Organic {
				if i >= 5 {
					break
				}
				title := item.Title
				if title == "" {
					title = item.Link
				}
				position := item.Position
				if position == 0 {
					position = i + 1
				}
				output.Bold.Printf("  [%d] %s\n", position, title)
				if item.Link != "" {
					fmt.Printf("      %s\n", item.Link)
				}
				if item.Snippet != "" {
					fmt.Printf("      %s\n", truncate(item.Snippet, 140))
				}
			}
			fmt.Println()
		}

		if len(result.SECFilings) > 0 {
			shown = true
			output.Bold.Println("  Latest SEC Filings:")
			for i, filing := range result.SECFilings {
				if i >= 3 {
					break
				}
				filedAt := filing.FiledAt
				if len(filedAt) >= 10 {
					filedAt = filedAt[:10]
				}
				output.KV(
					"Form", filing.FormType,
					"Filed", filedAt,
					"Company", filing.CompanyName,
					"Description", truncate(filing.Description, 120),
				)
				fmt.Println()
			}
		}

		if !shown {
			output.Warn.Println("  No displayable company details found. Use --json to view the full response.")
			fmt.Println()
		}

		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}

func truncate(s string, max int) string {
	return textutil.Truncate(s, max)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func joinNonEmpty(sep string, values ...string) string {
	var parts []string
	for _, value := range values {
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, sep)
}
