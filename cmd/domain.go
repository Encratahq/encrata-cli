package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var domainCmd = &cobra.Command{
	Use:   "domain [domain]",
	Short: "Look up a domain",
	Long:  "Retrieve WHOIS, DNS, SSL, and threat intelligence for a domain.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.New(cfg.BaseURL, cfg.APIKey)
		data, err := client.DomainSearch(cmd.Context(), args[0])
		if err != nil {
			output.Error(err.Error())
			return err
		}

		if cfg.Output == "json" {
			output.JSON(data)
			return nil
		}

		var result struct {
			Query  string `json:"query"`
			Domain string `json:"domain"`
			Whois  *struct {
				DomainName  string   `json:"domain_name"`
				Registrar   string   `json:"registrar"`
				CreatedDate string   `json:"created_date"`
				ExpiryDate  string   `json:"expiry_date"`
				Nameservers []string `json:"nameservers"`
			} `json:"whois"`
			SSL *struct {
				Valid    bool   `json:"valid"`
				Issuer   string `json:"issuer"`
				NotAfter string `json:"not_after"`
				DaysLeft int    `json:"days_left"`
				Protocol string `json:"protocol"`
			} `json:"ssl"`
			ThreatIntel *struct {
				Reputation int `json:"reputation"`
				Malicious  int `json:"malicious"`
				Suspicious int `json:"suspicious"`
				Harmless   int `json:"harmless"`
			} `json:"threat_intel"`
			Credits float64 `json:"credits"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		output.Header("Domain Lookup: " + args[0])

		if result.Whois != nil {
			w := result.Whois
			output.Bold.Println("  WHOIS:")
			output.KV(
				"Registrar", w.Registrar,
				"Created", w.CreatedDate,
				"Expires", w.ExpiryDate,
			)
			if len(w.Nameservers) > 0 {
				fmt.Printf("  %s  ", output.Bold.Sprint("Nameservers"))
				for i, ns := range w.Nameservers {
					if i > 0 {
						fmt.Print(", ")
					}
					fmt.Print(ns)
				}
				fmt.Println()
			}
			fmt.Println()
		}

		if result.SSL != nil {
			s := result.SSL
			valid := output.Err.Sprint("✗ Invalid")
			if s.Valid {
				valid = output.Success.Sprint("✓ Valid")
			}
			output.Bold.Println("  SSL:")
			output.KV(
				"Status", valid,
				"Issuer", s.Issuer,
				"Expires", s.NotAfter,
				"Days Left", fmt.Sprintf("%d", s.DaysLeft),
				"Protocol", s.Protocol,
			)
			fmt.Println()
		}

		if result.ThreatIntel != nil {
			t := result.ThreatIntel
			output.Bold.Println("  Threat Intel:")
			output.KV(
				"Reputation", fmt.Sprintf("%d", t.Reputation),
				"Malicious", fmt.Sprintf("%d", t.Malicious),
				"Suspicious", fmt.Sprintf("%d", t.Suspicious),
				"Harmless", fmt.Sprintf("%d", t.Harmless),
			)
			fmt.Println()
		}

		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}
