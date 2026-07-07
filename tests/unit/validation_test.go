package unit

import (
	"testing"

	"github.com/Encratahq/cli/internal/validation"
)

func TestTimeoutValidation(t *testing.T) {
	// Timeout accepts the documented range.
	for _, timeout := range []int{0, 1, 60000} {
		if err := validation.Timeout(timeout); err != nil {
			t.Fatalf("expected timeout %d to be valid: %v", timeout, err)
		}
	}

	// Timeout rejects values outside the documented range.
	for _, timeout := range []int{-1, 60001} {
		if err := validation.Timeout(timeout); err == nil {
			t.Fatalf("expected timeout %d to be invalid", timeout)
		}
	}
}

func TestThresholdValidation(t *testing.T) {
	// Threshold accepts the documented range.
	for _, threshold := range []float64{0, 0.5, 1} {
		if err := validation.Threshold(threshold); err != nil {
			t.Fatalf("expected threshold %.2f to be valid: %v", threshold, err)
		}
	}

	// Threshold rejects values outside the documented range.
	for _, threshold := range []float64{-0.1, 1.1} {
		if err := validation.Threshold(threshold); err == nil {
			t.Fatalf("expected threshold %.2f to be invalid", threshold)
		}
	}
}

func TestEmailValidation(t *testing.T) {
	for _, email := range []string{"user@example.com", "name+tag@example.co.in"} {
		if err := validation.Email(email); err != nil {
			t.Fatalf("expected email %q to be valid: %v", email, err)
		}
	}

	for _, email := range []string{"", "saiiran", "missing-domain@"} {
		if err := validation.Email(email); err == nil {
			t.Fatalf("expected email %q to be invalid", email)
		}
	}
}

func TestIPValidation(t *testing.T) {
	for _, address := range []string{"8.8.8.8", "2001:4860:4860::8888"} {
		if err := validation.IP(address); err != nil {
			t.Fatalf("expected IP %q to be valid: %v", address, err)
		}
	}

	for _, address := range []string{"", "saikiran", "999.999.999.999"} {
		if err := validation.IP(address); err == nil {
			t.Fatalf("expected IP %q to be invalid", address)
		}
	}
}

func TestDomainValidation(t *testing.T) {
	for _, domain := range []string{"example.com", "sub.example.co.in", "encrata.com."} {
		if err := validation.Domain(domain); err != nil {
			t.Fatalf("expected domain %q to be valid: %v", domain, err)
		}
	}

	for _, domain := range []string{"", "saikiran", "6303798093", "8.8.8.8", "https://example.com", "bad_domain.com", "-bad.com"} {
		if err := validation.Domain(domain); err == nil {
			t.Fatalf("expected domain %q to be invalid", domain)
		}
	}
}
