package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var emailIdentityCmd = &cobra.Command{
	Use:   "identity [email|file]",
	Short: "Person identity: name, role, company, socials, breaches (1 credit)",
	Long: `Look up the person, work history, education, and social profiles associated
with an email address.

Pass a single email to resolve it, or use --bulk with a file (or - for STDIN)
to resolve a whole list concurrently.

Examples:
  encrata email identity user@example.com
  encrata email identity emails.csv --bulk
  encrata email identity emails.csv --bulk --out people.csv`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bulk, _ := cmd.Flags().GetBool("bulk")
		if bulk {
			path := ""
			if len(args) == 1 {
				path = args[0]
			}
			return runIdentityBulk(cmd, path)
		}
		if len(args) != 1 {
			return friendlyFormatError(cmd, "provide one email address, or use --bulk with a file")
		}
		full, _ := cmd.Flags().GetBool("full")
		return emailLookup(cmd, args[0], "Identity", "Resolving identity...",
			(*api.Client).EmailIdentity,
			renderIdentity(full),
			nil)
	},
}

func init() {
	emailIdentityCmd.Flags().Bool("full", false, "Show the full breach list")
	emailIdentityCmd.Flags().Bool("bulk", false, "Resolve a file (or - for STDIN) of emails concurrently")
	emailIdentityCmd.Flags().Int("concurrency", 8, "Parallel lookups in --bulk mode")
	emailIdentityCmd.Flags().String("format", "", "Bulk export format: csv, xlsx, or json (default: inferred from --out)")
	emailIdentityCmd.Flags().String("only", "", "Export only rows matching: found")
	emailIdentityCmd.Flags().Bool("found-only", false, "Deprecated: use --only found")
	_ = emailIdentityCmd.Flags().MarkHidden("found-only")
}

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

// runIdentityBulk resolves identities for a file/STDIN list concurrently,
// prints a summary count, and optionally exports the rows.
func runIdentityBulk(cmd *cobra.Command, path string) error {
	if err := validateOnly(cmd, "found"); err != nil {
		return err
	}
	_, emails, _, err := loadEmails(cmd, path)
	if err != nil {
		return err
	}
	client, err := newClient()
	if err != nil {
		return err
	}
	out, _ := cmd.Flags().GetString("out")

	concurrency, _ := cmd.Flags().GetInt("concurrency")
	if concurrency < 1 {
		concurrency = 1
	}
	total := len(emails)
	if concurrency > total && total > 0 {
		concurrency = total
	}

	asJSON := jsonMode()
	if !asJSON {
		output.Header(fmt.Sprintf("Bulk Identity: %d %s", total, plural(total, "email", "emails")))
	}

	results := make([]map[string]interface{}, total)
	rawResults := make([]json.RawMessage, total)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var done int32
	var mu sync.Mutex

	for i, email := range emails {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, addr string) {
			defer wg.Done()
			defer func() { <-sem }()

			data, err := client.EmailIdentity(cmd.Context(), addr)
			row := map[string]interface{}{"email": addr}
			if err != nil {
				row["error"] = err.Error()
			} else if m, ok := decodeRow(data); ok {
				row = m
				if _, hasEmail := row["email"]; !hasEmail {
					row["email"] = addr
				}
				copyRaw := make(json.RawMessage, len(data))
				copy(copyRaw, data)
				rawResults[idx] = copyRaw
			}
			results[idx] = row

			if !asJSON {
				mu.Lock()
				done++
				renderProgress(int(done), total)
				mu.Unlock()
			}
		}(i, email)
	}
	wg.Wait()

	cleanRaw := make([]json.RawMessage, 0, total)
	for _, raw := range rawResults {
		if len(raw) > 0 {
			cleanRaw = append(cleanRaw, raw)
		}
	}

	found := countFoundIdentities(results)

	if asJSON {
		output.JSON(rawResultsArray(cleanRaw))
		if out != "" {
			return exportIdentity(cmd, out, results)
		}
		return nil
	}

	fmt.Println()
	fmt.Println()
	fmt.Printf("  Checked %d · Found %s · Not found %s\n",
		total,
		output.Success.Sprintf("%d", found),
		output.Dim.Sprintf("%d", total-found),
	)
	if out != "" {
		return exportIdentity(cmd, out, results)
	}
	return nil
}

// identityPerson returns the nested person object, or the row itself.
func identityPerson(r map[string]interface{}) map[string]interface{} {
	if p := getMap(r, "person"); p != nil {
		return p
	}
	return r
}

// identityFound reports whether a row resolved to a real person.
func identityFound(r map[string]interface{}) bool {
	if boolField(r, "found") == "true" {
		return true
	}
	p := identityPerson(r)
	if personName(p) != "" {
		return true
	}
	if field(p, "company", "pdl.job_company_name", "company_profile.name", "company_info.name") != "" {
		return true
	}
	return len(firstArr(p, "social_profiles", "socials")) > 0
}

// countFoundIdentities tallies rows that resolved to a person.
func countFoundIdentities(rows []map[string]interface{}) int {
	n := 0
	for _, r := range rows {
		if identityFound(r) {
			n++
		}
	}
	return n
}

// identityExportColumns is the flat schema for bulk-identity CSV/XLSX exports.
var identityExportColumns = []exportColumn{
	{"email", func(r map[string]interface{}) string { return field(r, "email", "query") }},
	{"found", func(r map[string]interface{}) string { return yesNo(strconv.FormatBool(identityFound(r))) }},
	{"name", func(r map[string]interface{}) string { return personName(identityPerson(r)) }},
	{"company", func(r map[string]interface{}) string {
		return field(identityPerson(r), "company", "pdl.job_company_name", "company_profile.name", "company_info.name")
	}},
	{"job_role", func(r map[string]interface{}) string {
		return field(identityPerson(r), "job_role", "job_title", "pdl.job_title", "title")
	}},
	{"location", func(r map[string]interface{}) string { return personLocation(identityPerson(r)) }},
}

// exportIdentity writes bulk-identity results to a file, honoring --format or
// the --out extension. JSON emits the raw, nested objects.
func exportIdentity(cmd *cobra.Command, out string, results []map[string]interface{}) error {
	_, _, foundOnly, _ := resolveOnlyFilters(cmd)
	rows := results
	if foundOnly {
		filtered := make([]map[string]interface{}, 0, len(results))
		for _, r := range results {
			if identityFound(r) {
				filtered = append(filtered, r)
			}
		}
		rows = filtered
	}

	formatFlag, _ := cmd.Flags().GetString("format")
	format, err := resolveExportFormat(formatFlag, out)
	if err != nil {
		return friendlyFormatError(cmd, err.Error())
	}
	if strings.TrimSpace(out) == "" && (format == "csv" || format == "xlsx") {
		out = fmt.Sprintf("email-identity.%s", format)
	}

	switch format {
	case "json":
		err = writeRawJSON(out, rows)
	case "xlsx":
		err = writeXLSX(out, identityExportColumns, rows)
	default:
		err = writeFlatCSV(out, identityExportColumns, rows)
	}
	if err != nil {
		return err
	}
	if info, statErr := os.Stat(out); statErr != nil || info.Size() == 0 {
		return fmt.Errorf("failed to write results to %s", out)
	}
	abs, absErr := filepath.Abs(out)
	if absErr != nil {
		abs = out
	}
	fmt.Fprintf(os.Stderr, "  Wrote %d %s\n", len(rows), plural(len(rows), "row", "rows"))
	output.SavedPath(abs)
	return nil
}
