package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
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
			Credits float64 `json:"credits"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		output.Header("Company Lookup: " + args[0])

		if result.Company == nil {
			output.Warn.Println("  No company data found")
			return nil
		}

		c := result.Company
		employees := ""
		if c.Employees > 0 {
			employees = fmt.Sprintf("%d", c.Employees)
		}

		output.KV(
			"Name", c.Name,
			"Industry", c.Industry,
			"HQ", c.Headquarter,
			"Website", c.Website,
			"Employees", employees,
			"Description", truncate(c.Description, 100),
		)

		if len(c.Specialties) > 0 {
			fmt.Println()
			output.Bold.Println("  Specialties:")
			for _, s := range c.Specialties {
				fmt.Printf("    • %s\n", s)
			}
		}

		fmt.Println()
		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
