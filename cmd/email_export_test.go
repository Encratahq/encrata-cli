package cmd

import (
	"regexp"
	"testing"
)

func TestNormalizedValidityStatus(t *testing.T) {
	cases := map[string]string{
		"valid":         "valid",
		"deliverable":   "valid",
		"invalid":       "invalid",
		"undeliverable": "invalid",
		"catch_all":     "catch-all",
		"accept-all":    "catch-all",
		"risky":         "risky",
	}

	for in, want := range cases {
		if got := normalizedValidityStatus(in); got != want {
			t.Fatalf("normalizedValidityStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDefaultValidityDownloadName(t *testing.T) {
	name := defaultValidityDownloadName("csv")
	if ok, _ := regexp.MatchString(`^email-validity-\d{4}-\d{2}-\d{2}\.csv$`, name); !ok {
		t.Fatalf("unexpected default filename: %q", name)
	}
}
