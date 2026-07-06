package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var emailCmd = &cobra.Command{
	Use:   "email [address]",
	Short: "Look up an email address",
	Long:  "Retrieve intelligence data for an email address including social profiles, breaches, and more.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.New(cfg.BaseURL, cfg.APIKey)

		country, _ := cmd.Flags().GetString("country")
		lang, _ := cmd.Flags().GetString("lang")
		num, _ := cmd.Flags().GetInt("num")
		page, _ := cmd.Flags().GetInt("page")
		fields, _ := cmd.Flags().GetStringSlice("fields")
		nocache, _ := cmd.Flags().GetBool("nocache")

		req := &api.EmailRequest{
			Email:   args[0],
			Country: country,
			Lang:    lang,
			Num:     num,
			Page:    page,
			Fields:  fields,
		}

		data, err := client.EmailLookup(cmd.Context(), req, nocache)
		if err != nil {
			output.Error(err.Error())
			return err
		}

		if cfg.Output == "json" {
			output.JSON(data)
			return nil
		}

		// Parse and display key fields
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		output.Header("Email Lookup: " + args[0])
		displayEmailResult(result)
		return nil
	},
}

func displayEmailResult(result map[string]interface{}) {
	credits := ""
	if c, ok := result["credits"]; ok {
		credits = fmt.Sprintf("%.0f", c)
	}

	if person, ok := result["person"].(map[string]interface{}); ok {
		name := getStr(person, "name")
		location := getStr(person, "location")
		bio := getStr(person, "bio")
		output.KV(
			"Name", name,
			"Location", location,
			"Bio", bio,
		)
		fmt.Println()
	}

	if socials, ok := result["social_profiles"].([]interface{}); ok && len(socials) > 0 {
		output.Bold.Println("  Social Profiles:")
		for _, s := range socials {
			if profile, ok := s.(map[string]interface{}); ok {
				platform := getStr(profile, "platform")
				url := getStr(profile, "url")
				fmt.Printf("    • %s: %s\n", platform, url)
			}
		}
		fmt.Println()
	}

	if breaches, ok := result["breaches"].([]interface{}); ok && len(breaches) > 0 {
		output.Warn.Printf("  ⚠ Found in %d breach(es)\n", len(breaches))
		for _, b := range breaches {
			if breach, ok := b.(map[string]interface{}); ok {
				name := getStr(breach, "name")
				date := getStr(breach, "date")
				fmt.Printf("    • %s (%s)\n", name, date)
			}
		}
		fmt.Println()
	}

	if credits != "" {
		output.Dim.Printf("  Credits used: %s\n", credits)
	}
}

func getStr(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func init() {
	emailCmd.Flags().String("country", "", "Country code (e.g. us, in)")
	emailCmd.Flags().String("lang", "", "Language code (e.g. en)")
	emailCmd.Flags().Int("num", 0, "Number of results")
	emailCmd.Flags().Int("page", 0, "Page number")
	emailCmd.Flags().StringSlice("fields", nil, "Specific fields to return")
	emailCmd.Flags().Bool("nocache", false, "Bypass the cache and run a fresh lookup")
}
