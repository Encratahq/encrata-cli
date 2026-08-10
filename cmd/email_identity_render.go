package cmd

import (
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/output"
)

func renderIdentity(full bool) func(map[string]interface{}) {
	return func(r map[string]interface{}) {
		// The identity response nests the profile under "person"; fall back to the
		// top level for older/flat shapes.
		person := getMap(r, "person")
		if person == nil {
			person = r
		}

		printNonEmptyKV(
			"Name", personName(person),
			"Job title", field(person, "job_role", "job_title", "pdl.job_title", "title"),
			"Company", field(person, "company", "pdl.job_company_name", "company_profile.name", "company_info.name"),
			"Industry", field(person, "industry", "pdl.job_company_industry"),
			"Location", personLocation(person),
			"Website", field(person, "website"),
			"Bio", field(person, "bio"),
		)
		fmt.Println()

		renderSocials(person)
		renderWorkHistory(person)
		renderEducation(person)

		registered := countField(person, "registered_services.registered_count", "registered_services.services")
		printNonEmptyKV(
			"Registered services", registered,
			"Breaches", countField(person, "breach_info.breach_count", "breach_info.count", "breaches", "breach_count"),
		)
		if full {
			renderBreachTable(person)
		}
	}
}

// personName composes a display name from a person object, joining the split
// name parts when a single name field is absent.
func personName(person map[string]interface{}) string {
	if name := field(person, "name", "full_name"); name != "" {
		return name
	}
	parts := make([]string, 0, 3)
	for _, key := range []string{"first_name", "middle_name", "last_name"} {
		if v := field(person, key); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// personLocation builds a readable location from current_location, or city and
// country when that field is absent.
func personLocation(person map[string]interface{}) string {
	if loc := field(person, "current_location", "location"); loc != "" {
		return loc
	}
	parts := make([]string, 0, 2)
	for _, key := range []string{"city", "country"} {
		if v := field(person, key); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, ", ")
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
		if arr = firstArr(r, "pdl.experience", "work_history", "experience", "employment", "jobs"); len(arr) > 0 {
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
			firstNonEmpty(field(m, "company_name", "company", "name", "organization"), "—"),
			firstNonEmpty(period(m), "—"),
		})
	}
	output.Table([]string{"Title", "Company", "Period"}, rows)
	fmt.Println()
}

func renderEducation(sources ...map[string]interface{}) {
	var arr []interface{}
	for _, r := range sources {
		if arr = firstArr(r, "pdl.education", "education", "schools"); len(arr) > 0 {
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
		degree := firstNonEmpty(strings2(
			firstNonEmpty(field(m, "degree"), listField(m, "degrees")),
			firstNonEmpty(field(m, "field", "field_of_study"), listField(m, "majors")),
		), "—")
		rows = append(rows, []string{
			firstNonEmpty(field(m, "school_name", "school", "name", "institution"), "—"),
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
