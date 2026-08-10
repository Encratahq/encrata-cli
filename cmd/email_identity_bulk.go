package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

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
	if strings.TrimSpace(out) == "" {
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
