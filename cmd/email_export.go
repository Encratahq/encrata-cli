package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

func rowIsValid(r map[string]interface{}) bool {
	return normalizedValidityStatus(field(r, "validity", "status")) == "valid"
}

func rowIsInvalid(r map[string]interface{}) bool {
	return normalizedValidityStatus(field(r, "validity", "status")) == "invalid"
}

// onlyFlag returns the trimmed --only value, or "" when the flag is absent.
func onlyFlag(cmd *cobra.Command) string {
	v, _ := cmd.Flags().GetString("only")
	return strings.TrimSpace(v)
}

// resolveOnlyFilters merges the --only flag with the deprecated --valid-only /
// --found-only booleans into concrete filter flags. Unknown --only values are
// ignored here (validated at the command layer via validateOnly).
func resolveOnlyFilters(cmd *cobra.Command) (validOnly, invalidOnly, foundOnly, breachedOnly bool) {
	switch strings.ToLower(onlyFlag(cmd)) {
	case "valid":
		validOnly = true
	case "invalid":
		invalidOnly = true
	case "found":
		foundOnly = true
	case "breached":
		breachedOnly = true
	}
	if v, _ := cmd.Flags().GetBool("valid-only"); v {
		validOnly = true
	}
	if f, _ := cmd.Flags().GetBool("found-only"); f {
		foundOnly = true
	}
	return
}

// deprecateFilterFlags registers the legacy per-status boolean filters as hidden
// aliases for --only, so old scripts keep working while help shows one flag.
func deprecateFilterFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("valid-only", false, "Deprecated: use --only valid")
	cmd.Flags().Bool("found-only", false, "Deprecated: use --only found")
	cmd.Flags().Bool("breached", false, "Deprecated: use --only breached")
	_ = cmd.Flags().MarkHidden("valid-only")
	_ = cmd.Flags().MarkHidden("found-only")
	_ = cmd.Flags().MarkHidden("breached")
}

// validityOnlyStatus maps --only valid|invalid to a validity results status
// filter, returning "" for other values.
func validityOnlyStatus(cmd *cobra.Command) string {
	validOnly, invalidOnly, _, _ := resolveOnlyFilters(cmd)
	switch {
	case validOnly:
		return "valid"
	case invalidOnly:
		return "invalid"
	default:
		return ""
	}
}

func defaultValidityDownloadName(format string) string {
	ext := strings.ToLower(strings.TrimSpace(format))
	if ext == "" {
		ext = "csv"
	}
	return fmt.Sprintf("email-validity-%s.%s", time.Now().Format("2006-01-02"), ext)
}

// resultsDir is the default folder CLI result files are written to when the user
// does not pass an explicit --out.
const resultsDir = "encrata-cli-results"

// resultStem derives a safe filename stem from a source path/name: the base name
// without directory or extension, keeping only [A-Za-z0-9._-] and collapsing any
// other run of characters to a single '-'. Falls back to "encrata" when empty.
func resultStem(source string) string {
	base := filepath.Base(strings.TrimSpace(source))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	var b strings.Builder
	prevDash := false
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	stem := strings.Trim(b.String(), "-")
	if stem == "" {
		return "encrata"
	}
	return stem
}

// defaultResultPath returns encrata-cli-results/<source-stem>-result.<ext>, the
// auto-generated output path when no --out is given, named after the uploaded
// file (or job). format is csv|xlsx|json.
func defaultResultPath(source, format string) string {
	ext := strings.ToLower(strings.TrimSpace(format))
	if ext == "" {
		ext = "csv"
	}
	return filepath.Join(resultsDir, resultStem(source)+"-result."+ext)
}

// resolveResultOut returns the file to write results to: an explicit --out wins;
// otherwise (in non-JSON mode) it auto-generates a path under resultsDir named
// after source. JSON mode keeps STDOUT piping unless --out is set.
func resolveResultOut(cmd *cobra.Command, source string) string {
	if out := strings.TrimSpace(flagString(cmd, "out")); out != "" {
		return out
	}
	if jsonMode() {
		return ""
	}
	format, _ := resolveExportFormat(flagString(cmd, "format"), "")
	return defaultResultPath(source, format)
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
	validOnly, invalidOnly, foundOnly, _ := resolveOnlyFilters(cmd)
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
	if invalidOnly {
		filtered := make([]map[string]interface{}, 0, len(rows))
		for _, r := range rows {
			if rowIsInvalid(r) {
				filtered = append(filtered, r)
			}
		}
		rows = filtered
	}

	if strings.TrimSpace(out) == "" {
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
