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
			output.Error(err.Error())
			return err
		}

		if !asJSON {
			fmt.Println()
			output.Dim.Printf("  %d result(s)\n", count)
		}
		return nil
	},
}

func bulkSearchCmd(use, short string, fn func(*api.Client, context.Context, []string) (json.RawMessage, error)) *cobra.Command {
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
				output.Error(err.Error())
				return err
			}
			output.JSON(data)
			return nil
		},
	}
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

func init() {
	bulkLookupCmd.Flags().StringSlice("fields", nil, "Specific fields to return")
	bulkLookupCmd.Flags().StringP("file", "f", "", "Read inputs from a file (one per line)")

	google := bulkSearchCmd("google [query...]", "Run up to 100 Google searches", (*api.Client).BulkGoogleSearch)
	company := bulkSearchCmd("company [query...]", "Enrich up to 100 companies", (*api.Client).BulkCompanySearch)
	domain := bulkSearchCmd("domain [query...]", "Run intelligence on up to 100 domains", (*api.Client).BulkDomainSearch)
	ip := bulkSearchCmd("ip [query...]", "Run intelligence on up to 100 IPs", (*api.Client).BulkIPSearch)

	for _, c := range []*cobra.Command{google, company, domain, ip} {
		c.Flags().StringP("file", "f", "", "Read inputs from a file (one per line)")
		bulkCmd.AddCommand(c)
	}
	bulkCmd.AddCommand(bulkLookupCmd)
}
