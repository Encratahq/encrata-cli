package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// exportColumn is one flattened output column: a header name plus a resolver
// that pulls the value out of a (possibly nested) bulk result row.
type exportColumn struct {
	name    string
	resolve func(map[string]interface{}) string
}

// exportColumns is the full, ordered set of columns written to CSV/XLSX by
// default. Booleans render as yes/no and lists join with " | ". email, status
// and reason are always present; everything else is "" when not returned.
var exportColumns = []exportColumn{
	{"email", func(r map[string]interface{}) string { return field(r, "email") }},
	{"status", func(r map[string]interface{}) string { return normalizedValidityStatus(field(r, "validity", "status")) }},
	{"reason", func(r map[string]interface{}) string { return field(r, "reason") }},
	{"message", func(r map[string]interface{}) string { return field(r, "message") }},
	{"confidence", func(r map[string]interface{}) string { return field(r, "confidence", "signals.confidence", "score") }},
	{"disposable", func(r map[string]interface{}) string { return yesNo(boolField(r, "disposable", "signals.disposable")) }},
	{"role", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "role", "role_account", "signals.role"))
	}},
	{"role_name", func(r map[string]interface{}) string { return field(r, "role_name", "signals.role_name") }},
	{"free_provider", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "free_provider", "free", "signals.free_provider"))
	}},
	{"provider", func(r map[string]interface{}) string { return field(r, "provider", "signals.provider", "esp") }},
	{"did_you_mean", func(r map[string]interface{}) string { return field(r, "did_you_mean", "suggestion", "didyoumean") }},
	{"canonical", func(r map[string]interface{}) string { return field(r, "canonical", "canonical_email", "normalized") }},
	{"domain", func(r map[string]interface{}) string { return field(r, "domain") }},
	{"mx", func(r map[string]interface{}) string { return pipeList(r, "mx", "mx_records", "signals.mx") }},
	{"smtp_mx_host", func(r map[string]interface{}) string { return field(r, "smtp.mx_host", "mx_host", "smtp.host") }},
	{"smtp_catch_all", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "smtp.catch_all", "catch_all", "catchall"))
	}},
	{"smtp_greylisted", func(r map[string]interface{}) string { return yesNo(boolField(r, "smtp.greylisted", "greylisted")) }},
	{"trust_grade", func(r map[string]interface{}) string {
		return field(r, "domain_trust.grade", "trust.grade", "trust_grade")
	}},
	{"spf", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "domain_trust.spf", "trust.spf", "spf"))
	}},
	{"dmarc", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "domain_trust.dmarc", "trust.dmarc", "dmarc"))
	}},
	{"dmarc_policy", func(r map[string]interface{}) string {
		return field(r, "domain_trust.dmarc_policy", "trust.dmarc_policy", "dmarc_policy")
	}},
	{"dkim", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "domain_trust.dkim", "trust.dkim", "dkim"))
	}},
	{"mta_sts", func(r map[string]interface{}) string {
		return field(r, "domain_trust.mta_sts", "trust.mta_sts", "mta_sts")
	}},
	{"tls_rpt", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "domain_trust.tls_rpt", "trust.tls_rpt", "tls_rpt"))
	}},
	{"bimi", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "domain_trust.bimi", "trust.bimi", "bimi"))
	}},
	{"dnssec", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "domain_trust.dnssec", "domain_info.dnssec", "trust.dnssec", "dnssec"))
	}},
	{"person_signal_count", func(r map[string]interface{}) string {
		return firstNonEmpty(field(r, "person_signal.count"), countField(r, "person_signal.sources"))
	}},
	{"person_signal_sources", func(r map[string]interface{}) string { return pipeList(r, "person_signal.sources") }},
	{"registrar", func(r map[string]interface{}) string {
		return field(r, "domain_info.registrar", "registrar", "whois.registrar")
	}},
	{"domain_created_at", func(r map[string]interface{}) string { return field(r, "domain_info.created_at") }},
	{"domain_age_days", func(r map[string]interface{}) string { return field(r, "domain_info.age_days") }},
	{"breaches_count", func(r map[string]interface{}) string {
		return firstNonEmpty(field(r, "footprint.breaches.count", "breach_count"), countField(r, "footprint.breaches", "breaches"))
	}},
	{"gravatar", func(r map[string]interface{}) string {
		return flagIfPresent(r, "footprint", "footprint.gravatar_url", "footprint.gravatar_profile", "footprint.gravatar")
	}},
	{"registered_services", func(r map[string]interface{}) string {
		return countField(r, "footprint.registered_services", "registered_services")
	}},
	{"google_account", func(r map[string]interface{}) string {
		return flagIfPresent(r, "footprint", "footprint.google", "footprint.google_account", "google_account")
	}},
	{"checked_at", func(r map[string]interface{}) string { return field(r, "checked_at", "timestamp") }},
}

// normalizedValidityStatus maps API-specific status labels into the canonical
// export values: valid | invalid | catch-all | risky.
func normalizedValidityStatus(status string) string {
	s := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(status, "_", "-")))
	s = strings.ReplaceAll(s, " ", "-")
	switch s {
	case "valid", "deliverable", "safe", "ok":
		return "valid"
	case "invalid", "undeliverable", "failed", "bad":
		return "invalid"
	case "catch-all", "catchall", "accept-all":
		return "catch-all"
	case "risky", "risk":
		return "risky"
	default:
		return s
	}
}

func rowIsValid(r map[string]interface{}) bool {
	return normalizedValidityStatus(field(r, "validity", "status")) == "valid"
}

func defaultValidityDownloadName(format string) string {
	ext := strings.ToLower(strings.TrimSpace(format))
	if ext == "" {
		ext = "csv"
	}
	return fmt.Sprintf("email-validity-%s.%s", time.Now().Format("2006-01-02"), ext)
}

// alwaysColumns are included in every export regardless of --columns.
var alwaysColumns = map[string]bool{"email": true, "status": true, "reason": true}

// yesNo maps a boolField result ("true"/"false"/"") to yes/no/"".
func yesNo(b string) string {
	switch b {
	case "true":
		return "yes"
	case "false":
		return "no"
	default:
		return ""
	}
}

// pipeList joins the first array value found across keys with " | ", or returns
// the first non-empty scalar.
func pipeList(r map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		v, ok := lookupRaw(r, k)
		if !ok {
			continue
		}
		if arr, ok := v.([]interface{}); ok {
			parts := make([]string, 0, len(arr))
			for _, item := range arr {
				if item == nil {
					continue
				}
				if s := strings.TrimSpace(fmt.Sprintf("%v", item)); s != "" {
					parts = append(parts, s)
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, " | ")
			}
			continue
		}
		if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" {
			return s
		}
	}
	return ""
}

// truthy reports whether a decoded JSON value is a meaningful "present" value.
func truthy(v interface{}) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		s := strings.TrimSpace(t)
		return s != "" && !strings.EqualFold(s, "false")
	case float64:
		return t != 0
	default:
		return true
	}
}

// flagIfPresent renders yes/no when the presence object exists, else "".
// yes when any of keys resolves to a truthy value; otherwise no.
func flagIfPresent(r map[string]interface{}, presence string, keys ...string) string {
	if getMap(r, presence) == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := lookupRaw(r, k); ok && truthy(v) {
			return "yes"
		}
	}
	return "no"
}

// selectExportColumns returns the ordered columns to write. When requested is
// empty, all columns are returned; otherwise only requested ones (plus the
// always-included email/status/reason), preserving the canonical order.
func selectExportColumns(requested []string) []exportColumn {
	if len(requested) == 0 {
		return exportColumns
	}
	want := make(map[string]bool, len(requested)+len(alwaysColumns))
	for k := range alwaysColumns {
		want[k] = true
	}
	for _, c := range requested {
		want[strings.ToLower(strings.TrimSpace(c))] = true
	}
	cols := make([]exportColumn, 0, len(want))
	for _, c := range exportColumns {
		if want[c.name] {
			cols = append(cols, c)
		}
	}
	return cols
}

// rowIsEnriched reports whether a result carries any data beyond the basic
// email/status/reason/message fields.
func rowIsEnriched(r map[string]interface{}) bool {
	for _, c := range exportColumns {
		switch c.name {
		case "email", "status", "reason", "message":
			continue
		}
		if strings.TrimSpace(c.resolve(r)) != "" {
			return true
		}
	}
	return false
}

// resolveExportFormat determines the output format from an explicit --format
// flag, falling back to the --out file extension, then CSV.
func resolveExportFormat(format, out string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "csv", "xlsx", "json":
		return strings.ToLower(strings.TrimSpace(format)), nil
	case "":
		switch strings.ToLower(filepath.Ext(out)) {
		case ".json":
			return "json", nil
		case ".xlsx":
			return "xlsx", nil
		default:
			return "csv", nil
		}
	default:
		return "", fmt.Errorf("format must be csv, xlsx, or json")
	}
}

// exportBulk writes bulk results to a file, honoring --columns, --found-only
// and --format (or the --out extension). JSON emits the raw, nested objects.
func exportBulk(cmd *cobra.Command, out string, results []map[string]interface{}) error {
	columns, _ := cmd.Flags().GetStringSlice("columns")
	if len(columns) == 0 {
		columns, _ = cmd.Flags().GetStringSlice("fields")
	}
	foundOnly, _ := cmd.Flags().GetBool("found-only")
	validOnly, _ := cmd.Flags().GetBool("valid-only")
	formatFlag, _ := cmd.Flags().GetString("format")

	format, err := resolveExportFormat(formatFlag, out)
	if err != nil {
		return friendlyFormatError(cmd, err.Error())
	}

	rows := results
	if foundOnly {
		filtered := make([]map[string]interface{}, 0, len(results))
		for _, r := range results {
			if rowIsEnriched(r) {
				filtered = append(filtered, r)
			}
		}
		rows = filtered
	}
	if validOnly {
		filtered := make([]map[string]interface{}, 0, len(rows))
		for _, r := range rows {
			if rowIsValid(r) {
				filtered = append(filtered, r)
			}
		}
		rows = filtered
	}

	if strings.TrimSpace(out) == "" && (format == "csv" || format == "xlsx") {
		out = defaultValidityDownloadName(format)
	}

	switch format {
	case "json":
		err = writeRawJSON(out, rows)
	case "xlsx":
		err = writeXLSX(out, selectExportColumns(columns), rows)
	default:
		err = writeFlatCSV(out, selectExportColumns(columns), rows)
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

// writeFlatCSV writes a header row plus one flattened row per result.
func writeFlatCSV(path string, cols []exportColumn, rows []map[string]interface{}) error {
	data, err := buildFlatCSV(cols, rows)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// buildFlatCSV returns the flattened CSV payload (header + rows).
func buildFlatCSV(cols []exportColumn, rows []map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := make([]string, len(cols))
	for i, c := range cols {
		header[i] = c.name
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, r := range rows {
		record := make([]string, len(cols))
		for i, c := range cols {
			record[i] = c.resolve(r)
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeRawJSON writes the raw, nested result objects (unflattened).
func writeRawJSON(path string, rows []map[string]interface{}) error {
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// --- Minimal XLSX writer (single sheet, inline strings, no dependencies) ---

const (
	xlsxContentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
		`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
		`</Types>`

	xlsxRootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
		`</Relationships>`

	xlsxWorkbook = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
		`<sheets><sheet name="Results" sheetId="1" r:id="rId1"/></sheets></workbook>`

	xlsxWorkbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
		`</Relationships>`

	xlsxSheetOpen = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`
)

var xlsxEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&apos;",
)

// writeXLSX writes a single-sheet .xlsx with a header row and one row per result.
func writeXLSX(path string, cols []exportColumn, rows []map[string]interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	add := func(name, content string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, content)
		return err
	}

	if err := add("[Content_Types].xml", xlsxContentTypes); err != nil {
		return err
	}
	if err := add("_rels/.rels", xlsxRootRels); err != nil {
		return err
	}
	if err := add("xl/workbook.xml", xlsxWorkbook); err != nil {
		return err
	}
	if err := add("xl/_rels/workbook.xml.rels", xlsxWorkbookRels); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString(xlsxSheetOpen)
	sb.WriteString("<sheetData>")
	header := make([]string, len(cols))
	for i, c := range cols {
		header[i] = c.name
	}
	writeXLSXRow(&sb, 1, header)
	for i, r := range rows {
		values := make([]string, len(cols))
		for j, c := range cols {
			values[j] = c.resolve(r)
		}
		writeXLSXRow(&sb, i+2, values)
	}
	sb.WriteString("</sheetData></worksheet>")

	if err := add("xl/worksheets/sheet1.xml", sb.String()); err != nil {
		return err
	}
	return zw.Close()
}

func writeXLSXRow(sb *strings.Builder, rowNum int, values []string) {
	fmt.Fprintf(sb, `<row r="%d">`, rowNum)
	for i, v := range values {
		ref := xlsxColRef(i) + strconv.Itoa(rowNum)
		fmt.Fprintf(sb, `<c r="%s" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, xlsxEscaper.Replace(v))
	}
	sb.WriteString("</row>")
}

// xlsxColRef converts a 0-based column index to a spreadsheet letter (A, B, …, AA).
func xlsxColRef(i int) string {
	ref := ""
	for n := i + 1; n > 0; n /= 26 {
		n--
		ref = string(rune('A'+n%26)) + ref
	}
	return ref
}
