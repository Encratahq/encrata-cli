package api

import (
	"context"
	"encoding/json"
)

// EmailValidity checks whether a single email address is valid and deliverable,
// returning the validity report itself. The email service answers this route in
// the {success, result, message} envelope (ENC-2763); the report is unwrapped
// here, with the server's message folded in, so every caller (the single lookup,
// --enrich, --json, --out) reads the same bare report it always has.
// POST /api/cli/email/validity
func (c *Client) EmailValidity(ctx context.Context, email string) (json.RawMessage, error) {
	data, err := c.post(ctx, "/api/cli/email/validity", map[string]string{"email": email})
	if err != nil {
		return nil, err
	}
	return unwrapResult(data), nil
}

// EmailIdentity resolves the identity behind an email address.
// POST /api/cli/email/identity
func (c *Client) EmailIdentity(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/cli/email/identity", map[string]string{"email": email})
}

// EmailBreaches lists known data breaches an email appears in.
// POST /api/cli/email/breaches
func (c *Client) EmailBreaches(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/cli/email/breaches", map[string]string{"email": email})
}

// EmailCompliance answers whether an address may be cold-emailed and under whose
// law, returning a jurisdiction, its rule, and the confidence of the attribution.
// POST /api/cli/email/compliance
func (c *Client) EmailCompliance(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/cli/email/compliance", map[string]string{"email": email})
}

// EmailValidityBulk validates a batch of emails in a single synchronous request
// (payload up to 64MB).
// POST /api/cli/email/validity/bulk
func (c *Client) EmailValidityBulk(ctx context.Context, emails []string) (json.RawMessage, error) {
	return c.post(ctx, "/api/cli/email/validity/bulk", map[string][]string{"emails": emails})
}
