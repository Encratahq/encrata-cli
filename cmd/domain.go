package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

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
			DNS *struct {
				A     []string `json:"a"`
				AAAA  []string `json:"aaaa"`
				MX    []string `json:"mx"`
				NS    []string `json:"ns"`
				TXT   []string `json:"txt"`
				CNAME string   `json:"cname"`
			} `json:"dns"`

			Intel *struct {
				Subdomains []string `json:"subdomains"`
				IPs        []string `json:"ips"`
				Tech       []string `json:"tech"`
			} `json:"intel"`

			Company *struct {
				Name         string `json:"name"`
				Domain       string `json:"domain"`
				Description  string `json:"description"`
				Industry     string `json:"industry"`
				LegalName    string `json:"legal_name"`
				Founded      string `json:"founded"`
				Headquarters string `json:"headquarters"`
			} `json:"company"`
			Extras *struct {
				Popularity *struct {
					Rank int    `json:"rank"`
					List string `json:"list"`
					Date string `json:"date"`
				} `json:"popularity"`
				Typosquat *struct {
					Count      int `json:"count"`
					Generated  int `json:"generated"`
					Registered []struct {
						Domain string `json:"domain"`
						Type   string `json:"type"`
						IP     string `json:"ip"`
					} `json:"registered"`
				} `json:"typosquat"`
				URLScan *struct {
					Malicious  bool   `json:"malicious"`
					ReportURL  string `json:"report_url"`
					Screenshot string `json:"screenshot"`
				} `json:"urlscan"`
			} `json:"extras"`
			Report *struct {
				Summary *struct {
					CompanyName string   `json:"company_name"`
					Category    string   `json:"category"`
					Confidence  string   `json:"confidence"`
					KeyFindings []string `json:"key_findings"`
					MajorRisks  []string `json:"major_risks"`
					Stats       struct {
						Subdomains   int `json:"subdomains"`
						DNSRecords   int `json:"dns_records"`
						LiveHosts    int `json:"live_hosts"`
						OpenPorts    int `json:"open_ports"`
						Technologies int `json:"technologies"`
					} `json:"stats"`
				} `json:"summary"`
				Risk []struct {
					Signal         string `json:"signal"`
					Severity       string `json:"severity"`
					Category       string `json:"category"`
					Recommendation string `json:"recommendation"`
				} `json:"risk"`
				Security []struct {
					Area           string `json:"area"`
					Finding        string `json:"finding"`
					Severity       string `json:"severity"`
					Recommendation string `json:"recommendation"`
				} `json:"security"`
				Screenshot string `json:"screenshot"`
			} `json:"report"`
			Credits float64 `json:"credits"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		output.Header("Domain Lookup: " + args[0])
		shown := false
		if result.Whois != nil {
			shown = true
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
			shown = true
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
			shown = true
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

		if result.DNS != nil {
			shown = true
			d := result.DNS
			output.Bold.Println("  DNS:")
			output.KV(
				"A", strings.Join(d.A, ", "),
				"AAAA", strings.Join(d.AAAA, ", "),
				"MX", strings.Join(d.MX, ", "),
				"NS", strings.Join(d.NS, ", "),
				"TXT", strings.Join(d.TXT, ", "),
				"CNAME", d.CNAME,
			)
			fmt.Println()
		}

		if result.Intel != nil {
			shown = true
			i := result.Intel
			output.Bold.Println("  Intelligence:")
			output.KV(
				"Subdomains", strings.Join(i.Subdomains, ", "),
				"IPs", strings.Join(i.IPs, ", "),
				"Technology", strings.Join(i.Tech, ", "),
			)
			fmt.Println()
		}

		if result.Company != nil {
			shown = true
			c := result.Company
			output.Bold.Println("  Company:")
			output.KV(
				"Name", c.Name,
				"Legal Name", c.LegalName,
				"Domain", c.Domain,
				"Founded", c.Founded,
				"HQ", c.Headquarters,
				"Description", c.Description,
				"Industry", c.Industry,
			)
			fmt.Println()
		}

		if result.Report != nil && result.Report.Summary != nil {
			shown = true
			s := result.Report.Summary
			output.Bold.Println("  Summary:")
			output.KV(
				"Company", s.CompanyName,
				"Category", s.Category,
				"Confidence", s.Confidence,
				"Subdomains", fmt.Sprintf("%d", s.Stats.Subdomains),
				"DNS Records", fmt.Sprintf("%d", s.Stats.DNSRecords),
				"Live Hosts", fmt.Sprintf("%d", s.Stats.LiveHosts),
			)
			if len(s.KeyFindings) > 0 {
				fmt.Printf("  %s  %s\n", output.Bold.Sprint("Findings"), strings.Join(s.KeyFindings, "; "))
			}
			if len(s.MajorRisks) > 0 {
				fmt.Printf("  %s  %s\n", output.Bold.Sprint("Risks"), strings.Join(s.MajorRisks, "; "))
			}
			fmt.Println()
		}

		if result.Extras != nil {
			if result.Extras.Popularity != nil || result.Extras.Typosquat != nil || result.Extras.URLScan != nil {
				shown = true
				output.Bold.Println("  Extras:")
				if p := result.Extras.Popularity; p != nil {
					output.KV("Popularity", fmt.Sprintf("%s #%d", p.List, p.Rank), "Rank Date", p.Date)
				}
				if t := result.Extras.Typosquat; t != nil {
					output.KV("Typosquats", fmt.Sprintf("%d registered / %d generated", t.Count, t.Generated))
					for i, r := range t.Registered {
						if i >= 5 {
							break
						}
						fmt.Printf("    %s (%s", r.Domain, r.Type)
						if r.IP != "" {
							fmt.Printf(", %s", r.IP)
						}
						fmt.Println(")")
					}
				}
				if u := result.Extras.URLScan; u != nil {
					status := output.Success.Sprint("Clean")
					if u.Malicious {
						status = output.Err.Sprint("Malicious")
					}
					output.KV("URLScan", status, "Report", u.ReportURL)
				}
				fmt.Println()
			}
		}

		if result.Report != nil && len(result.Report.Risk) > 0 {
			shown = true
			output.Bold.Println("  Risks:")
			for i, r := range result.Report.Risk {
				if i >= 5 {
					break
				}
				output.KV(
					"Signal", r.Signal,
					"Severity", r.Severity,
					"Category", r.Category,
					"Recommendation", r.Recommendation,
				)
				fmt.Println()
			}
		}

		if result.Report != nil && len(result.Report.Security) > 0 {
			shown = true
			output.Bold.Println("  Security:")
			for i, s := range result.Report.Security {
				if i >= 5 {
					break
				}
				output.KV(
					"Area", s.Area,
					"Finding", s.Finding,
					"Severity", s.Severity,
				)
			}
			fmt.Println()
		}

		if !shown {
			output.Warn.Println("  No displayable domain details found. Use --json to view the full response.")
			fmt.Println()
		}

		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}
