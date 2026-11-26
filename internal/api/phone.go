package api

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Phone lookups are owned by the lookup service and reached through the CLI
// proxy at /api/cli/phone/<op> (forwarded to /api/lookup/phone/<op>). Every
// lookup answers in the {success, result, message} envelope; the methods below
// return the unwrapped result so callers render one shape.

// Phone sub-lookups served at /api/cli/phone/<feature> with a {phone} body.
const (
	PhoneFeatureValidity = "validity"
	PhoneFeatureRisk     = "risk"
	PhoneFeatureBreaches = "breaches"
	PhoneFeatureIdentity = "identity"
)

// PhoneFeatures is the set of {phone}-body sub-lookups.
var PhoneFeatures = []string{PhoneFeatureValidity, PhoneFeatureRisk, PhoneFeatureBreaches, PhoneFeatureIdentity}

// IsPhoneFeature reports whether f is a {phone}-body sub-lookup.
func IsPhoneFeature(f string) bool {
	for _, x := range PhoneFeatures {
		if f == x {
			return true
		}
	}
	return false
}

var (
	phoneFormatting = regexp.MustCompile(`[\s().\-/]`)
	phoneE164       = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
)

// NormalizePhone turns a human-formatted international number into the strict
// E.164 form the server accepts (^\+[1-9][0-9]{1,14}$): spaces, dashes, dots,
// slashes and parentheses are dropped, and a leading 00 becomes +. A number
// without a country code is rejected rather than guessed. The server still runs
// the authoritative validity check.
func NormalizePhone(raw string) (string, error) {
	s := phoneFormatting.ReplaceAllString(strings.TrimSpace(raw), "")
	if strings.HasPrefix(s, "00") {
		s = "+" + s[2:]
	}
	if s == "" {
		return "", fmt.Errorf("provide a phone number")
	}
	if !strings.HasPrefix(s, "+") {
		return "", fmt.Errorf("%q needs a country code, in international format (e.g. +14155552671)", strings.TrimSpace(raw))
	}
	if !phoneE164.MatchString(s) {
		return "", fmt.Errorf("%q is not a valid international phone number (e.g. +14155552671)", strings.TrimSpace(raw))
	}
	return s, nil
}

// PhoneSearch runs the full phone report. include optionally restricts the
// sections (format, country, carrier, line_type, messaging, validation,
// registration, risk, breaches).
// POST /api/cli/phone/search {"query": "+1...", "include": [...]}
func (c *Client) PhoneSearch(ctx context.Context, phone string, include []string) (json.RawMessage, error) {
	payload := map[string]interface{}{"query": phone}
	if len(include) > 0 {
		payload["include"] = include
	}
	data, err := c.post(ctx, "/api/cli/phone/search", payload)
	if err != nil {
		return nil, err
	}
	return unwrapResult(data), nil
}

// PhoneLookup runs one phone sub-lookup (validity | risk | breaches | identity).
// POST /api/cli/phone/<feature> {"phone": "+1..."}
func (c *Client) PhoneLookup(ctx context.Context, feature, phone string) (json.RawMessage, error) {
	if !IsPhoneFeature(feature) {
		return nil, fmt.Errorf("unknown phone lookup %q", feature)
	}
	data, err := c.post(ctx, "/api/cli/phone/"+feature, map[string]string{"phone": phone})
	if err != nil {
		return nil, err
	}
	return unwrapResult(data), nil
}

// PhoneVoip runs the VoIP-line check. mode is "static" (default) or "current",
// which also resolves number portability.
// POST /api/cli/phone/voip {"phone": "+1...", "mode": "current"}
func (c *Client) PhoneVoip(ctx context.Context, phone, mode string) (json.RawMessage, error) {
	payload := map[string]string{"phone": phone}
	if mode != "" {
		payload["mode"] = mode
	}
	data, err := c.post(ctx, "/api/cli/phone/voip", payload)
	if err != nil {
		return nil, err
	}
	return unwrapResult(data), nil
}
