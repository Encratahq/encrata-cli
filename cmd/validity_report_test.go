package cmd

import (
	"encoding/json"
	"testing"
)

const reportJSON = `{"email":"a@b.com","domain":"b.com","status":"undeliverable","validity":"invalid",
"outcome":{"reason":"mailbox_not_found","message":"The server said the mailbox does not exist.","cause":"mailbox","ip_blocked":true},
"conclusion":{"has_mx":true,"is_catch_all":false,"is_deliverable":false,"is_mailbox_full":false},
"quota":{"is_full":false},
"dns":{"mx":[{"host":"mx1.b.com","preference":10},{"host":"mx2.b.com","preference":20}],"spf":"v=spf1 -all","dmarc":null},
"connection":{"mx_host":"mx1.b.com"},
"extra":{"provider":{"name":"Google"},"score":{"score":12,"grade":"F"},"syntax":{"disposable":false,"role":true,"free_provider":false},
"dns_deep":{"dkim":[{"selector":"google"}],"dmarc":{"policy":"none"}}},
"credits":1}`

func exportRow(t *testing.T, r map[string]interface{}) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, c := range exportColumns {
		out[c.name] = c.resolve(r)
	}
	return out
}

// A smtpprobe report fills the export columns from its own blocks.
func TestExportColumnsReadTheReport(t *testing.T) {
	var r map[string]interface{}
	if err := json.Unmarshal([]byte(reportJSON), &r); err != nil {
		t.Fatal(err)
	}
	got := exportRow(t, r)
	want := map[string]string{
		"status": "invalid", "deliverability": "undeliverable", "reason": "mailbox_not_found",
		"message": "The server said the mailbox does not exist.", "score": "12", "score_grade": "F",
		"mailbox_full": "no", "role": "yes", "disposable": "no", "provider": "Google",
		"mx": "mx1.b.com | mx2.b.com", "smtp_mx_host": "mx1.b.com", "smtp_catch_all": "no",
		"spf": "yes", "dmarc": "no", "dmarc_policy": "none", "dkim": "yes",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if !isValidityReport(r) || unverified(r) {
		t.Error("report detection is wrong")
	}
	if scoreLabel(r) != "12 (F)" || mxHosts(r) != "mx1.b.com, mx2.b.com" {
		t.Errorf("score %q, mx %q", scoreLabel(r), mxHosts(r))
	}
}

// A job row keeps its own four-bucket status; the report's lands in deliverability.
func TestExportColumnsReadAJobRow(t *testing.T) {
	row := map[string]interface{}{"email": "a@b.com", "status": "catch-all", "report_status": "unknown", "reason": "accepted"}
	got := exportRow(t, row)
	if got["status"] != "catch-all" || got["deliverability"] != "unknown" || got["reason"] != "accepted" {
		t.Fatalf("row = %v", got)
	}
}

// An older result (no report blocks) leaves the new columns blank.
func TestExportColumnsLegacyResult(t *testing.T) {
	row := map[string]interface{}{"email": "a@b.com", "status": "valid", "domain_trust": map[string]interface{}{"spf": true}}
	got := exportRow(t, row)
	if got["deliverability"] != "" || got["score"] != "" || got["spf"] != "yes" || got["status"] != "valid" {
		t.Fatalf("row = %v", got)
	}
}

func TestUnverifiedReport(t *testing.T) {
	r := map[string]interface{}{"conclusion": map[string]interface{}{}, "outcome": map[string]interface{}{"cause": "us"}}
	if !unverified(r) {
		t.Fatal("cause us must read as unverified")
	}
	if humanize("mailbox_not_found") != "Mailbox not found" {
		t.Fatal("humanize")
	}
}
