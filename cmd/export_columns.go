package cmd

import (
	"strings"
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

// alwaysColumns are included in every export regardless of --columns.
var alwaysColumns = map[string]bool{"email": true, "status": true, "reason": true}

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
