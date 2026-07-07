package validation

import (
	"fmt"
	"net/mail"
	"net/netip"
	"strings"
)

func Email(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email address is required")
	}
	if _, err := mail.ParseAddress(email); err != nil || !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email address")
	}
	return nil
}

func IP(address string) error {
	address = strings.TrimSpace(address)
	if address == "" {
		return fmt.Errorf("IP address is required")
	}
	if _, err := netip.ParseAddr(address); err != nil {
		return fmt.Errorf("invalid IP address")
	}
	return nil
}

func Domain(domain string) error {
	domain = strings.TrimSpace(strings.TrimSuffix(domain, "."))
	if domain == "" {
		return fmt.Errorf("domain is required")
	}
	if strings.Contains(domain, "://") || strings.ContainsAny(domain, "/?#@:") {
		return fmt.Errorf("invalid domain")
	}
	if _, err := netip.ParseAddr(domain); err == nil {
		return fmt.Errorf("expected a domain, not an IP address")
	}
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return fmt.Errorf("invalid domain")
	}
	for _, label := range labels {
		if !validDomainLabel(label) {
			return fmt.Errorf("invalid domain")
		}
	}
	tld := labels[len(labels)-1]
	if allDigits(tld) || len(tld) < 2 {
		return fmt.Errorf("invalid domain")
	}
	return nil
}

func validDomainLabel(label string) bool {
	if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return false
	}
	for _, r := range label {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func Timeout(timeout int) error {
	if timeout < 0 || timeout > 60000 {
		return fmt.Errorf("timeout must be between 0 and 60000 milliseconds")
	}
	return nil
}

func Threshold(threshold float64) error {
	if threshold < 0 || threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1")
	}
	return nil
}
