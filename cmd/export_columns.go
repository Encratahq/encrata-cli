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
	{"reason", func(r map[string]interface{}) string { return field(r, "reason", "outcome.reason", "status_reason") }},
	{"message", func(r map[string]interface{}) string { return field(r, "message", "outcome.message") }},
	// The smtpprobe report's own status (deliverable | undeliverable | unknown),
	// its score and mailbox-full flag; blank for older results.
	{"deliverability", func(r map[string]interface{}) string { return reportStatus(r) }},
	{"score", func(r map[string]interface{}) string { return field(r, "extra.score.score") }},
	{"score_grade", func(r map[string]interface{}) string { return field(r, "extra.score.grade") }},
	{"mailbox_full", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "quota.is_full", "conclusion.is_mailbox_full"))
	}},
	{"confidence", func(r map[string]interface{}) string { return field(r, "confidence", "signals.confidence", "score") }},
	{"disposable", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "disposable", "signals.disposable", "extra.syntax.disposable"))
	}},
	{"role", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "role", "role_account", "signals.role", "extra.syntax.role"))
	}},
	{"role_name", func(r map[string]interface{}) string { return field(r, "role_name", "signals.role_name") }},
	{"free_provider", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "free_provider", "free", "signals.free_provider", "extra.syntax.free_provider"))
	}},
	{"provider", func(r map[string]interface{}) string {
		return field(r, "provider", "signals.provider", "esp", "extra.provider.name")
	}},
	{"did_you_mean", func(r map[string]interface{}) string { return field(r, "did_you_mean", "suggestion", "didyoumean") }},
	{"canonical", func(r map[string]interface{}) string {
		return field(r, "canonical", "canonical_email", "normalized", "extra.syntax.normalized")
	}},
	{"domain", func(r map[string]interface{}) string { return field(r, "domain") }},
	{"mx", func(r map[string]interface{}) string {
		return firstNonEmpty(pipeList(r, "mx", "mx_records", "signals.mx"), strings.ReplaceAll(mxHosts(r), ", ", " | "))
	}},
	{"smtp_mx_host", func(r map[string]interface{}) string {
		return field(r, "smtp.mx_host", "mx_host", "smtp.host", "connection.mx_host")
	}},
	{"smtp_catch_all", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "smtp.catch_all", "catch_all", "catchall", "conclusion.is_catch_all"))
	}},
	{"smtp_greylisted", func(r map[string]interface{}) string { return yesNo(boolField(r, "smtp.greylisted", "greylisted")) }},
	{"trust_grade", func(r map[string]interface{}) string {
		return field(r, "domain_trust.grade", "trust.grade", "trust_grade")
	}},
	{"spf", func(r map[string]interface{}) string {
		return firstNonEmpty(yesNo(boolField(r, "domain_trust.spf", "trust.spf", "spf")), dnsRecord(r, "spf"))
	}},
	{"dmarc", func(r map[string]interface{}) string {
		return firstNonEmpty(yesNo(boolField(r, "domain_trust.dmarc", "trust.dmarc", "dmarc")), dnsRecord(r, "dmarc"))
	}},
	{"dmarc_policy", func(r map[string]interface{}) string {
		return field(r, "domain_trust.dmarc_policy", "trust.dmarc_policy", "dmarc_policy", "extra.dns_deep.dmarc.policy")
	}},
	{"dkim", func(r map[string]interface{}) string {
		return firstNonEmpty(yesNo(boolField(r, "domain_trust.dkim", "trust.dkim", "dkim")), dkimFound(r))
	}},
	{"mta_sts", func(r map[string]interface{}) string {
		return field(r, "domain_trust.mta_sts", "trust.mta_sts", "mta_sts", "dns.mta_sts")
	}},
	{"tls_rpt", func(r map[string]interface{}) string {
		return yesNo(boolField(r, "domain_trust.tls_rpt", "trust.tls_rpt", "tls_rpt"))
	}},
	{"bimi", func(r map[string]interface{}) string {
		return firstNonEmpty(yesNo(boolField(r, "domain_trust.bimi", "trust.bimi", "bimi")), dnsRecord(r, "bimi"))
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

// reportStatus is a smtpprobe report's own status: report_status on a job row
// (whose status is the four-bucket verdict), or status on a single report.
func reportStatus(r map[string]interface{}) string {
	if s := field(r, "report_status"); s != "" {
		return s
	}
	if isValidityReport(r) {
		return field(r, "status")
	}
	return ""
}

// dnsRecord is yes/no for a smtpprobe report's DNS record (dns.<name> holds the
// record text, null when absent), or "" when the row has no DNS block.
func dnsRecord(r map[string]interface{}, name string) string {
	dns, ok := r["dns"].(map[string]interface{})
	if !ok {
		return ""
	}
	if s, _ := dns[name].(string); strings.TrimSpace(s) != "" {
		return "yes"
	}
	return "no"
}

// dkimFound is yes/no for the DKIM selectors a report found, or "" when the
// report did no deep DNS check.
func dkimFound(r map[string]interface{}) string {
	v, ok := lookupRaw(r, "extra.dns_deep.dkim")
	if !ok {
		return ""
	}
	if arr, _ := v.([]interface{}); len(arr) > 0 {
		return "yes"
	}
	return "no"
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
