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

func defaultValidityDownloadName(format string) string {
	ext := strings.ToLower(strings.TrimSpace(format))
	if ext == "" {
		ext = "csv"
	}
	return fmt.Sprintf("email-validity-%s.%s", time.Now().Format("2006-01-02"), ext)
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
