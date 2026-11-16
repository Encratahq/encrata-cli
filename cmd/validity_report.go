package cmd

import (
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/output"
)

// The smtpprobe validity report (ENC-2752): what `email validity` returns once
// the email service checks addresses with a live mail-server probe. Its own
// status is deliverable | undeliverable | unknown; `validity` keeps the older
// four-bucket verdict (valid | invalid | catch-all | risky) that bulk jobs count.
// Older results have none of outcome / conclusion / quota and render with the
// legacy view (renderFullSections).

// isValidityReport reports whether a validity response is a smtpprobe report.
func isValidityReport(r map[string]interface{}) bool {
	_, ok := r["conclusion"].(map[string]interface{})
	return ok
}

// unverified reports whether the mail server refused our check (outcome cause
// "us"): the address was not tested, so it is never shown as invalid.
func unverified(r map[string]interface{}) bool {
	return strings.EqualFold(field(r, "outcome.cause"), "us")
}

// renderValidityReport prints the compact verdict of a report, and every detail
// section with full. Admin-only outcome fields (ip_blocked, blocked_by,
// client_ip) are never printed.
func renderValidityReport(r map[string]interface{}, full bool) {
	printNonEmptyKV(
		"Email", field(r, "email"),
		"Status", statusColor(field(r, "status")),
		"Verdict", statusColor(field(r, "validity")),
		"Reason", humanize(field(r, "outcome.reason", "status_reason", "reason")),
		"Catch-all", yesNo(boolField(r, "conclusion.is_catch_all")),
		"Mailbox full", yesNo(boolField(r, "quota.is_full", "conclusion.is_mailbox_full")),
		"Disposable", yesNo(boolField(r, "extra.syntax.disposable")),
		"Role account", yesNo(boolField(r, "extra.syntax.role")),
		"Free provider", yesNo(boolField(r, "extra.syntax.free_provider")),
		"Provider", field(r, "extra.provider.name"),
		"Score", scoreLabel(r),
	)
	if msg := field(r, "outcome.message", "message"); msg != "" {
		fmt.Println()
		output.Dim.Println("  " + msg)
	}
	if unverified(r) {
		output.Warn.Println("  The mail server refused our check, so this address could not be verified. It is not invalid; try again later.")
	}
	if full {
		renderValidityReportSections(r)
	}
}

// renderValidityReportSections prints the report's detail blocks.
func renderValidityReportSections(r map[string]interface{}) {
	fmt.Println()
	section("Outcome",
		"Stage", humanize(field(r, "outcome.stage")),
		"Code", joinNonEmpty(" ", field(r, "outcome.code"), field(r, "outcome.enhanced_code")),
		"Permanent", yesNo(boolField(r, "outcome.permanent")),
		"Retryable", yesNo(boolField(r, "outcome.retryable")),
		"Server reply", field(r, "outcome.reply"),
	)
	section("DNS",
		"MX", mxHosts(r),
		"SPF", field(r, "dns.spf"),
		"DMARC", field(r, "dns.dmarc"),
		"BIMI", field(r, "dns.bimi"),
		"MTA-STS", field(r, "dns.mta_sts"),
	)
	section("Connection",
		"MX host", field(r, "connection.mx_host"),
		"IP", joinNonEmpty(" ", field(r, "connection.ip"), parens(field(r, "connection.ip_family"))),
		"Reverse DNS", field(r, "connection.reverse_dns"),
		"Banner", field(r, "connection.banner"),
		"Connect (ms)", field(r, "connection.connect_ms"),
	)
	section("SMTP",
		"RCPT code", joinNonEmpty(" ", field(r, "smtp.rcpt_code"), field(r, "smtp.rcpt_enhanced_code")),
		"RCPT reply", field(r, "smtp.rcpt_reply"),
		"STARTTLS", yesNo(boolField(r, "smtp.supports_starttls")),
		"Catch-all", yesNo(boolField(r, "smtp.catch_all")),
		"Max message size", field(r, "smtp.max_size_bytes"),
	)
	section("TLS",
		"Version", field(r, "tls.tls_version"),
		"Cipher", field(r, "tls.cipher.name"),
		"Certificate issuer", field(r, "tls.certificate.issuer"),
		"Certificate expires", timeField(r, "tls.certificate.not_after"),
	)
	section("Catch-all test",
		"Accepted", fraction(field(r, "extra.catch_all.accepted"), field(r, "extra.catch_all.total")),
		"Confidence", humanize(field(r, "extra.catch_all.confidence")),
	)
	section("Blocklists",
		"Listed", fraction(field(r, "extra.blocklist.listed"), field(r, "extra.blocklist.checked")),
	)
	section("Score",
		"Score", scoreLabel(r),
		"Reasons", listField(r, "extra.score.reasons"),
	)
	printNonEmptyKV(
		"Checked at", timeField(r, "checked_at"),
		"Cached", boolField(r, "cached"),
	)
}

// validityFooter prints the credit line, noting a free check when the mail
// server never answered (only an answered check is charged).
func validityFooter(r map[string]interface{}) {
	if v, ok := r["provider_answered"].(bool); ok && !v {
		output.Dim.Println("  Free — the mail server didn't answer, so nothing was charged")
		return
	}
	printCredits(r)
}

// scoreLabel is "92 (A)", "92", or "" when the report has no score.
func scoreLabel(r map[string]interface{}) string {
	return joinNonEmpty(" ", field(r, "extra.score.score"), parens(field(r, "extra.score.grade")))
}

// mxHosts lists the report's MX hosts in preference order.
func mxHosts(r map[string]interface{}) string {
	hosts := make([]string, 0, 4)
	for _, item := range firstArr(r, "dns.mx") {
		switch t := item.(type) {
		case map[string]interface{}:
			if h := getStr(t, "host"); h != "" {
				hosts = append(hosts, h)
			}
		case string:
			hosts = append(hosts, t)
		}
	}
	return strings.Join(hosts, ", ")
}

// humanize turns "mailbox_not_found" into "Mailbox not found".
func humanize(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "_", " "))
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func parens(s string) string {
	if s == "" {
		return ""
	}
	return "(" + s + ")"
}

// fraction is "3 of 5", or "" when either side is missing.
func fraction(n, of string) string {
	if n == "" || of == "" {
		return ""
	}
	return n + " of " + of
}

func joinNonEmpty(sep string, parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}
