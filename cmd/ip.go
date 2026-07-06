package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var ipCmd = &cobra.Command{
	Use:   "ip [address]",
	Short: "Look up an IP address",
	Long:  "Retrieve geolocation, ASN, company, and threat data for an IP address.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.New(cfg.BaseURL, cfg.APIKey)
		data, err := client.IPSearch(cmd.Context(), args[0])
		if err != nil {
			output.Error(err.Error())
			return err
		}

		if cfg.Output == "json" {
			output.JSON(data)
			return nil
		}

		var result struct {
			Query    string `json:"query"`
			IP       string `json:"ip"`
			Location *struct {
				City        string `json:"city"`
				Region      string `json:"region"`
				Country     string `json:"country"`
				CountryCode string `json:"country_code"`
				PostalCode  string `json:"postal_code"`
			} `json:"location"`
			ASN *struct {
				Number int    `json:"number"`
				Org    string `json:"org"`
				ISP    string `json:"isp"`
				Type   string `json:"type"`
			} `json:"asn"`
			Company *struct {
				Name   string `json:"name"`
				Domain string `json:"domain"`
				Type   string `json:"type"`
			} `json:"company"`
			Threat *struct {
				IsTor    bool `json:"is_tor"`
				IsProxy  bool `json:"is_proxy"`
				IsVPN    bool `json:"is_vpn"`
				IsAbuser bool `json:"is_abuser"`
				IsBot    bool `json:"is_bot"`
			} `json:"threat"`
			Credits float64 `json:"credits"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		output.Header("IP Lookup: " + args[0])

		// Location
		if result.Location != nil {
			l := result.Location
			output.KV(
				"City", l.City,
				"Region", l.Region,
				"Country", fmt.Sprintf("%s (%s)", l.Country, l.CountryCode),
				"Postal", l.PostalCode,
			)
			fmt.Println()
		}

		// ASN
		if result.ASN != nil {
			a := result.ASN
			output.Bold.Println("  Network:")
			output.KV(
				"ASN", fmt.Sprintf("AS%d", a.Number),
				"Org", a.Org,
				"ISP", a.ISP,
				"Type", a.Type,
			)
			fmt.Println()
		}

		// Company
		if result.Company != nil {
			c := result.Company
			output.Bold.Println("  Company:")
			output.KV(
				"Name", c.Name,
				"Domain", c.Domain,
				"Type", c.Type,
			)
			fmt.Println()
		}

		// Threat
		if result.Threat != nil {
			t := result.Threat
			output.Bold.Println("  Threat Assessment:")
			threats := []struct {
				label string
				val   bool
			}{
				{"Tor", t.IsTor},
				{"Proxy", t.IsProxy},
				{"VPN", t.IsVPN},
				{"Abuser", t.IsAbuser},
				{"Bot", t.IsBot},
			}
			for _, th := range threats {
				indicator := output.Success.Sprint("✓ No")
				if th.val {
					indicator = output.Err.Sprint("⚠ Yes")
				}
				fmt.Printf("    %-8s %s\n", th.label, indicator)
			}
			fmt.Println()
		}

		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}
