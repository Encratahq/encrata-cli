package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var emailIdentityCmd = &cobra.Command{
	Use:   "identity [email]",
	Short: "Resolve the identity behind an email address",
	Long:  "Look up the person, work history, education, and social profiles associated with an email address.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		full, _ := cmd.Flags().GetBool("full")
		return emailLookup(cmd, args[0], "Identity", "Resolving identity...",
			(*api.Client).EmailIdentity,
			renderIdentity(full),
			nil)
	},
}

func init() {
	emailIdentityCmd.Flags().Bool("full", false, "Show the full breach list")
}

func renderIdentity(full bool) func(map[string]interface{}) {
	return func(r map[string]interface{}) {
		person := getMap(r, "person")
		if person == nil {
			person = r
		}
		printNonEmptyKV(
			"Name", field(r, "name", "full_name", "person.name"),
			"Job title", field(r, "job_title", "title", "person.job_title", "person.title"),
			"Company", field(r, "company", "company.name", "person.company", "person.company.name"),
			"Location", field(r, "location", "person.location"),
		)
		fmt.Println()

		renderSocials(r, person)
		renderWorkHistory(r, person)
		renderEducation(r, person)

		printNonEmptyKV("Breaches", countField(r, "breaches", "breach_count", "footprint.breaches"))
		if full {
			renderBreachTable(r)
		}
	}
}

func renderSocials(sources ...map[string]interface{}) {
	printed := false
	for _, r := range sources {
		if socials := getMap(r, "socials"); len(socials) > 0 {
			if !printed {
				output.Bold.Println("  Socials:")
				printed = true
			}
			for platform, url := range socials {
				if v := fmt.Sprintf("%v", url); v != "" && v != "<nil>" {
					fmt.Printf("    %s: %s\n", platform, v)
				}
			}
		}
		for _, s := range firstArr(r, "social_profiles", "socials") {
			m := asMap(s)
			platform := field(m, "platform", "network", "type")
			url := field(m, "url", "link")
			if platform == "" && url == "" {
				continue
			}
			if !printed {
				output.Bold.Println("  Socials:")
				printed = true
			}
			fmt.Printf("    %s: %s\n", firstNonEmpty(platform, "—"), firstNonEmpty(url, "—"))
		}
	}
	if printed {
		fmt.Println()
	}
}

func renderWorkHistory(sources ...map[string]interface{}) {
	var arr []interface{}
	for _, r := range sources {
		if arr = firstArr(r, "work_history", "experience", "employment", "jobs"); len(arr) > 0 {
			break
		}
	}
	if len(arr) == 0 {
		return
	}
	output.Bold.Println("  Work history:")
	rows := make([][]string, 0, len(arr))
	for _, it := range arr {
		m := asMap(it)
		rows = append(rows, []string{
			firstNonEmpty(field(m, "title", "role", "position"), "—"),
			firstNonEmpty(field(m, "company", "name", "organization"), "—"),
			firstNonEmpty(period(m), "—"),
		})
	}
	output.Table([]string{"Title", "Company", "Period"}, rows)
	fmt.Println()
}

func renderEducation(sources ...map[string]interface{}) {
	var arr []interface{}
	for _, r := range sources {
		if arr = firstArr(r, "education", "schools"); len(arr) > 0 {
			break
		}
	}
	if len(arr) == 0 {
		return
	}
	output.Bold.Println("  Education:")
	rows := make([][]string, 0, len(arr))
	for _, it := range arr {
		m := asMap(it)
		degree := firstNonEmpty(strings2(field(m, "degree"), field(m, "field", "field_of_study")), "—")
		rows = append(rows, []string{
			firstNonEmpty(field(m, "school", "name", "institution"), "—"),
			degree,
			firstNonEmpty(period(m), "—"),
		})
	}
	output.Table([]string{"School", "Degree / field", "Period"}, rows)
	fmt.Println()
}

// strings2 joins a degree and field with " / " when both are present.
func strings2(a, b string) string {
	switch {
	case a != "" && b != "":
		return a + " / " + b
	case a != "":
		return a
	default:
		return b
	}
}
