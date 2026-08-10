package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/Encratahq/cli/internal/output"
)

// statusColor colors a status value using the shared palette. Empty input
// returns "" so callers render a "—" placeholder.
func statusColor(status string) string {
	trimmed := strings.TrimSpace(status)
	if trimmed == "" {
		return ""
	}
	switch strings.ToLower(strings.ReplaceAll(trimmed, "_", "-")) {
	case "valid", "deliverable", "safe", "ok":
		return output.Success.Sprint(trimmed)
	case "invalid", "undeliverable", "failed", "bad":
		return output.Err.Sprint(trimmed)
	case "catch-all", "catchall", "accept-all":
		return output.Warn.Sprint(trimmed) // amber
	case "risky", "risk":
		return output.Brand.Sprint(trimmed) // orange
	case "unknown":
		return output.Dim.Sprint(trimmed) // grey
	default:
		return trimmed
	}
}

// deliverableLabel maps a validity status to yes/no, or "" when indeterminate.
func deliverableLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "valid", "deliverable":
		return "yes"
	case "invalid", "undeliverable":
		return "no"
	default:
		return ""
	}
}

// timeField parses common timestamp formats and renders them in local time.
func timeField(r map[string]interface{}, keys ...string) string {
	s := field(r, keys...)
	if s == "" {
		return ""
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if ts, err := time.Parse(layout, s); err == nil {
			return ts.Local().Format("Jan 2, 2006 3:04 PM")
		}
	}
	return s
}

// monthYear renders a date-ish string as "Jan 2006".
func monthYear(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01", "2006/01", "01/2006", "2006"} {
		if ts, err := time.Parse(layout, s); err == nil {
			return ts.Format("Jan 2006")
		}
	}
	return s
}

// period formats an employment/education period as "Mon YYYY → Mon YYYY"/"Present".
func period(m map[string]interface{}) string {
	start := monthYear(field(m, "start", "start_date", "from", "starts_at"))
	end := field(m, "end", "end_date", "to", "ends_at")
	endLabel := monthYear(end)
	if boolField(m, "current", "is_current") == "true" || strings.EqualFold(strings.TrimSpace(end), "present") || end == "" {
		endLabel = "Present"
	}
	if start == "" && (endLabel == "" || endLabel == "Present") {
		return ""
	}
	return fmt.Sprintf("%s → %s", firstNonEmpty(start, "—"), firstNonEmpty(endLabel, "—"))
}
