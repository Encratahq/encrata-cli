package cmd

import (
	"fmt"
	"strconv"
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

// lookupRaw resolves a flat key or a dotted "parent.child" path.
func lookupRaw(r map[string]interface{}, key string) (interface{}, bool) {
	if strings.Contains(key, ".") {
		parts := strings.SplitN(key, ".", 2)
		if child, ok := r[parts[0]].(map[string]interface{}); ok {
			return lookupRaw(child, parts[1])
		}
		return nil, false
	}
	v, ok := r[key]
	if !ok || v == nil {
		return nil, false
	}
	return v, true
}

// field returns the first non-empty string value across the given keys.
func field(r map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := lookupRaw(r, k); ok {
			switch t := v.(type) {
			case float64:
				return formatNumber(t)
			default:
				s := strings.TrimSpace(fmt.Sprintf("%v", v))
				if s != "" && s != "<nil>" {
					return s
				}
			}
		}
	}
	return ""
}

// boolField renders a present boolean as true/false, or "" when absent.
func boolField(r map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := lookupRaw(r, k); ok {
			switch t := v.(type) {
			case bool:
				return strconv.FormatBool(t)
			case string:
				s := strings.ToLower(strings.TrimSpace(t))
				if s == "true" || s == "false" {
					return s
				}
			}
		}
	}
	return ""
}

// listField joins an array value with ", ", or returns a scalar as-is.
func listField(r map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := lookupRaw(r, k); ok {
			if arr, ok := v.([]interface{}); ok {
				if len(arr) == 0 {
					continue
				}
				return joinInterfaces(arr)
			}
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

// countField returns the count of a list, or a numeric value as an integer.
func countField(r map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := lookupRaw(r, k); ok {
			switch t := v.(type) {
			case []interface{}:
				return strconv.Itoa(len(t))
			case float64:
				return strconv.Itoa(int(t))
			case string:
				if t != "" {
					return t
				}
			}
		}
	}
	return ""
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

// getMap returns a nested object, or nil.
func getMap(r map[string]interface{}, key string) map[string]interface{} {
	if v, ok := r[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

// firstArr returns the first present array across the given keys.
func firstArr(r map[string]interface{}, keys ...string) []interface{} {
	for _, k := range keys {
		if v, ok := lookupRaw(r, k); ok {
			if arr, ok := v.([]interface{}); ok {
				return arr
			}
		}
	}
	return nil
}

func asMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func formatNumber(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// creditsValue returns the credit cost from a response, defaulting to "0".
func creditsValue(r map[string]interface{}) string {
	if v := field(r, "credits", "credits_used", "credit"); v != "" {
		return v
	}
	return "0"
}

// intOf parses a numeric-ish string, defaulting to 0.
func intOf(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// --- Shared full-detail sections (validity --full / enrich) ---

// section prints a titled block of only the non-empty key/value pairs, and is
// skipped entirely when every value is empty.
func section(title string, pairs ...string) {
	filtered := make([]string, 0, len(pairs))
	for i := 0; i+1 < len(pairs); i += 2 {
		if strings.TrimSpace(pairs[i+1]) == "" {
			continue
		}
		filtered = append(filtered, pairs[i], pairs[i+1])
	}
	if len(filtered) == 0 {
		return
	}
	output.Bold.Println("  " + title + ":")
	output.KV(filtered...)
	fmt.Println()
}

func renderSignals(r map[string]interface{}) {
	section("Signals",
		"Confidence", field(r, "confidence", "signals.confidence", "score"),
		"Free provider", boolField(r, "free_provider", "free", "signals.free_provider"),
		"Did you mean", field(r, "did_you_mean", "suggestion", "didyoumean"),
		"Canonical", field(r, "canonical", "canonical_email", "normalized"),
		"Provider", field(r, "provider", "signals.provider", "esp"),
		"Domain", field(r, "domain"),
		"MX", listField(r, "mx", "mx_records", "signals.mx"),
	)
}

func renderSMTP(r map[string]interface{}) {
	section("SMTP",
		"MX host", field(r, "smtp.mx_host", "mx_host", "smtp.host"),
		"Code", field(r, "smtp.code", "smtp_code"),
		"Message", field(r, "smtp.message"),
		"Catch-all", boolField(r, "smtp.catch_all", "catch_all", "catchall"),
		"Greylisted", boolField(r, "smtp.greylisted", "greylisted"),
	)
}

func renderDomainTrust(r map[string]interface{}) {
	section("Domain trust",
		"Grade", field(r, "domain_trust.grade", "trust.grade", "trust_grade"),
		"SPF", boolField(r, "domain_trust.spf", "trust.spf", "spf"),
		"DMARC", boolField(r, "domain_trust.dmarc", "trust.dmarc", "dmarc"),
		"DMARC policy", field(r, "domain_trust.dmarc_policy", "trust.dmarc_policy", "dmarc_policy"),
		"DKIM", boolField(r, "domain_trust.dkim", "trust.dkim", "dkim"),
		"MTA-STS", field(r, "domain_trust.mta_sts", "trust.mta_sts", "mta_sts"),
		"TLS-RPT", boolField(r, "domain_trust.tls_rpt", "trust.tls_rpt", "tls_rpt"),
		"BIMI", boolField(r, "domain_trust.bimi", "trust.bimi", "bimi"),
		"DNSSEC", boolField(r, "domain_trust.dnssec", "domain_info.dnssec", "trust.dnssec", "dnssec"),
	)
}

func renderFootprint(r map[string]interface{}) {
	section("Footprint",
		"Breaches", countField(r, "footprint.breaches", "breaches", "breach_count"),
		"Gravatar", boolField(r, "footprint.gravatar", "gravatar"),
		"Registered services", countField(r, "footprint.registered_services", "registered_services"),
		"Google account", boolField(r, "footprint.google_account", "google_account"),
	)
}

func renderDomainInfo(r map[string]interface{}) {
	section("Domain info",
		"Registrar", field(r, "domain_info.registrar", "registrar", "whois.registrar"),
		"Domain created", timeField(r, "domain_info.created", "domain_created", "whois.created_date"),
		"Domain age (days)", field(r, "domain_info.age_days", "domain_age_days"),
	)
}

func renderPersonSignals(r map[string]interface{}) {
	count := intOf(countField(r, "person_signal.count", "person_signals", "person.signals"))
	if count <= 0 {
		return
	}
	section("Person signals",
		"Signals", strconv.Itoa(count),
		"Sources", listField(r, "person_signal.sources", "signal_sources", "person.signal_sources"),
	)
}

// renderFullSections renders the shared full-detail block for validity/enrich.
func renderFullSections(r map[string]interface{}) {
	fmt.Println()
	renderSignals(r)
	renderSMTP(r)
	renderDomainTrust(r)
	renderFootprint(r)
	renderDomainInfo(r)
	renderPersonSignals(r)
	printNonEmptyKV(
		"Checked at", timeField(r, "checked_at", "created_at", "timestamp"),
		"Cached", boolField(r, "cached"),
	)
}
