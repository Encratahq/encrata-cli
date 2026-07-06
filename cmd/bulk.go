package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var bulkCmd = &cobra.Command{
	Use:   "bulk",
	Short: "Bulk enrichment and search operations",
}

var bulkLookupCmd = &cobra.Command{
	Use:   "lookup [email...]",
	Short: "Enrich up to 1,000 emails (streamed)",
	Long:  "Enrich many emails in one request. Results stream back as they complete. Provide emails as arguments or via --file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		emails, err := collectInputs(cmd, args)
		if err != nil {
			return err
		}
		fields, _ := cmd.Flags().GetStringSlice("fields")

		asJSON := jsonMode()
		if !asJSON {
			output.Header(fmt.Sprintf("Bulk Lookup: %d email(s)", len(emails)))
		}

		count := 0
		err = client.BulkLookup(cmd.Context(), emails, fields, func(event json.RawMessage) error {
			count++
			if asJSON {
				output.JSON(event)
				return nil
			}
			var person map[string]interface{}
			if decode(event, &person) {
				printPersonLine(person)
			}
			return nil
		})
		if err != nil {

			return err
		}

		if !asJSON {
			fmt.Println()
			output.Dim.Printf("  %d result(s)\n", count)
		}
		return nil
	},
}

func bulkSearchCmd(
	use, short string,
	fn func(*api.Client, context.Context, []string) (json.RawMessage, error),
	render func(json.RawMessage) error,
) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			queries, err := collectInputs(cmd, args)
			if err != nil {
				return err
			}
			data, err := fn(client, cmd.Context(), queries)
			if err != nil {
				return err
			}
			if jsonMode() {
				output.JSON(data)
				return nil
			}
			return render(data)
		},
	}
}

func renderBulkCompanies(data json.RawMessage) error {
	var results []map[string]interface{}
	if !decode(data, &results) {
		output.JSON(data)
		return nil
	}

	output.Header(fmt.Sprintf("Bulk Company: %d result(s)", len(results)))

	totalCredits := 0.0
	for i, item := range results {
		query := getStr(item, "query")
		credits := getFloat(item, "credits")
		totalCredits += credits

		title := query
		if title == "" {
			title = fmt.Sprintf("Result %d", i+1)
		}
		output.Bold.Printf("  [%d] %s\n", i+1, title)

		if profile, ok := item["profile"].(map[string]interface{}); ok {
			output.KV(
				"Name", getStr(profile, "name"),
				"Industry", getStr(profile, "industry"),
				"Website", firstNonEmpty(getStr(profile, "website"), getStr(profile, "domain")),
			)
		} else if company, ok := item["company"].(map[string]interface{}); ok {
			output.KV(
				"Name", getStr(company, "name"),
				"Industry", getStr(company, "industry"),
				"Website", getStr(company, "website"),
			)
		} else if organic, ok := item["organic"].([]interface{}); ok && len(organic) > 0 {
			if first, ok := organic[0].(map[string]interface{}); ok {
				output.KV(
					"Top Result", getStr(first, "title"),
					"URL", getStr(first, "link"),
				)
			}
		} else {
			output.Warn.Println("  No displayable company details")
		}

		if credits > 0 {
			output.Dim.Printf("  Credits: %.0f\n", credits)
		}
		fmt.Println()
	}

	output.Dim.Printf("  Total credits used: %.0f\n", totalCredits)
	return nil
}

func renderBulkJSON(data json.RawMessage) error {
	output.JSON(data)
	return nil
}

func renderBulkSummaries(title string, data json.RawMessage) error {
	var results []map[string]interface{}
	if !decode(data, &results) {
		output.JSON(data)
		return nil
	}

	output.Header(fmt.Sprintf("%s: %d result(s)", title, len(results)))

	for i, item := range results {
		query := getStr(item, "query")
		if query == "" {
			query = fmt.Sprintf("Result %d", i+1)
		}

		output.Bold.Printf("  [%d] %s\n", i+1, query)

		if errMsg := getStr(item, "error"); errMsg != "" {
			output.KV("Error", errMsg)
			fmt.Println()
			continue
		}

		printBulkSummary(item)
		fmt.Println()
	}

	return nil
}

func printBulkSummary(item map[string]interface{}) {
	if _, hasIP := item["ip"]; hasIP {
		printBulkIPSummary(item)
		return
	}

	if whois, ok := item["whois"].(map[string]interface{}); ok {
		output.KV(
			"Domain", firstNonEmpty(getStr(whois, "domain_name"), getStr(item, "domain"), getStr(item, "query")),
			"Registrar", getStr(whois, "registrar"),
			"Created", getStr(whois, "created_date"),
			"Expires", getStr(whois, "expiry_date"),
		)
		return
	}

	if report, ok := item["report"].(map[string]interface{}); ok {
		if summary, ok := report["summary"].(map[string]interface{}); ok {
			output.KV(
				"Company", getStr(summary, "company_name"),
				"Category", getStr(summary, "category"),
				"Confidence", getStr(summary, "confidence"),
			)
			return
		}
	}

	if kg, ok := item["knowledge_graph"].(map[string]interface{}); ok {
		output.KV(
			"Title", getStr(kg, "title"),
			"Type", getStr(kg, "type"),
			"Website", getStr(kg, "website"),
			"Description", truncate(getStr(kg, "description"), 120),
		)
		return
	}

	if organic, ok := item["organic"].([]interface{}); ok && len(organic) > 0 {
		if first, ok := organic[0].(map[string]interface{}); ok {
			output.KV(
				"Top Result", getStr(first, "title"),
				"URL", getStr(first, "link"),
				"Snippet", truncate(getStr(first, "snippet"), 120),
			)
			return
		}
	}

	output.Warn.Println("  No displayable details. Use --json to view the full response.")
}

func printBulkIPSummary(item map[string]interface{}) {
	if loc, ok := item["location"].(map[string]interface{}); ok {
		country := getStr(loc, "country")
		if code := getStr(loc, "country_code"); code != "" {
			country = fmt.Sprintf("%s (%s)", country, code)
		}
		output.KV(
			"City", getStr(loc, "city"),
			"Region", getStr(loc, "region"),
			"Country", country,
			"Postal", getStr(loc, "postal_code"),
		)
	}

	if asn, ok := item["asn"].(map[string]interface{}); ok {
		number := getFloat(asn, "number")
		asnLabel := ""
		if number > 0 {
			asnLabel = fmt.Sprintf("AS%.0f", number)
		}
		output.Bold.Println("    Network:")
		output.KV(
			"ASN", asnLabel,
			"Org", getStr(asn, "org"),
			"ISP", getStr(asn, "isp"),
			"Type", getStr(asn, "type"),
		)
	}

	if company, ok := item["company"].(map[string]interface{}); ok {
		output.Bold.Println("    Company:")
		output.KV(
			"Name", getStr(company, "name"),
			"Domain", getStr(company, "domain"),
			"Type", getStr(company, "type"),
		)
	}

	if threat, ok := item["threat"].(map[string]interface{}); ok {
		output.Bold.Println("    Threat Assessment:")
		for _, row := range []struct {
			label string
			key   string
		}{
			{"Tor", "is_tor"},
			{"Proxy", "is_proxy"},
			{"VPN", "is_vpn"},
			{"Abuser", "is_abuser"},
			{"Bot", "is_bot"},
		} {
			fmt.Printf("      %-8s %s\n", row.label, yesNo(getBool(threat, row.key)))
		}
	}
}

func yesNo(v bool) string {
	if v {
		return output.Err.Sprint("Yes")
	}
	return output.Success.Sprint("No")
}

func renderBulkGoogle(data json.RawMessage) error {
	return renderBulkSummaries("Bulk Google", data)
}

func renderBulkDomains(data json.RawMessage) error {
	return renderBulkSummaries("Bulk Domain", data)
}

func renderBulkIPs(data json.RawMessage) error {
	return renderBulkSummaries("Bulk IP", data)
}

func collectInputs(cmd *cobra.Command, args []string) ([]string, error) {
	if file, _ := cmd.Flags().GetString("file"); file != "" {
		lines, err := readLines(file)
		if err != nil {
			return nil, err
		}
		args = append(args, lines...)
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("provide at least one value as an argument or via --file")
	}
	return args, nil
}

func printPersonLine(resp map[string]interface{}) {
	email := getStr(resp, "email")
	name := ""
	company := ""
	if person, ok := resp["person"].(map[string]interface{}); ok {
		name = strings.TrimSpace(getStr(person, "first_name") + " " + getStr(person, "last_name"))
		company = getStr(person, "company")
	}
	if name == "" {
		name = "—"
	}
	line := fmt.Sprintf("  • %s  %s", output.Bold.Sprint(name), output.Dim.Sprint(email))
	if company != "" {
		line += output.Dim.Sprint("  · " + company)
	}
	fmt.Println(line)
}

func getFloat(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

func init() {
	bulkLookupCmd.Flags().StringSlice("fields", nil, "Specific fields to return")
	bulkLookupCmd.Flags().StringP("file", "f", "", "Read inputs from a file (one per line)")

	google := bulkSearchCmd("google [query...]", "Run up to 100 Google searches", (*api.Client).BulkGoogleSearch, renderBulkGoogle)
	company := bulkSearchCmd("company [query...]", "Enrich up to 100 companies", (*api.Client).BulkCompanySearch, renderBulkCompanies)
	domain := bulkSearchCmd("domain [query...]", "Run intelligence on up to 100 domains", (*api.Client).BulkDomainSearch, renderBulkDomains)
	ip := bulkSearchCmd("ip [query...]", "Run intelligence on up to 100 IPs", (*api.Client).BulkIPSearch, renderBulkIPs)

	for _, c := range []*cobra.Command{google, company, domain, ip} {
		c.Flags().StringP("file", "f", "", "Read inputs from a file (one per line)")
		bulkCmd.AddCommand(c)
	}
	bulkCmd.AddCommand(bulkLookupCmd)
}
