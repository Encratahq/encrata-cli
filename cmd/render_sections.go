package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Encratahq/cli/internal/output"
)

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
		"Breaches", countField(r, "footprint.breaches.count", "footprint.breaches.breach_count", "breaches", "breach_count"),
		"Gravatar", presenceLabel(r, "footprint.gravatar_url", "footprint.gravatar_profile", "gravatar"),
		"Registered services", countField(r, "footprint.registered_services", "registered_services.registered_count", "registered_services"),
		"Google account", firstNonEmpty(field(r, "footprint.google.name"), presenceLabel(r, "footprint.google", "google_account")),
	)
}

// presenceLabel returns "yes" when any of the given keys resolves to a truthy,
// non-empty value (bool true, non-empty string / object / array).
func presenceLabel(r map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		v, ok := lookupRaw(r, k)
		if !ok {
			continue
		}
		switch t := v.(type) {
		case bool:
			if t {
				return "yes"
			}
		case string:
			if strings.TrimSpace(t) != "" {
				return "yes"
			}
		case map[string]interface{}:
			if len(t) > 0 {
				return "yes"
			}
		case []interface{}:
			if len(t) > 0 {
				return "yes"
			}
		}
	}
	return ""
}

func renderDomainInfo(r map[string]interface{}) {
	section("Domain info",
		"Registrar", field(r, "domain_info.registrar", "registrar", "whois.registrar"),
		"Domain created", timeField(r, "domain_info.created_at", "domain_info.created", "domain_created", "whois.created_date"),
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
