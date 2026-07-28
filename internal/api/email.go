package api

import (
	"context"
	"encoding/json"
)

// EmailValidity checks whether a single email address is valid and deliverable.
// POST /api/email/validity
func (c *Client) EmailValidity(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/email/validity", map[string]string{"email": email})
}

// EmailEnrich returns validity plus enrichment (person/company) data.
// POST /api/email/validity/enrich
func (c *Client) EmailEnrich(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/email/validity/enrich", map[string]string{"email": email})
}

// EmailIdentity resolves the identity behind an email address.
// POST /api/email/identity
func (c *Client) EmailIdentity(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/email/identity", map[string]string{"email": email})
}

// EmailBreaches lists known data breaches an email appears in.
// POST /api/email/breaches
func (c *Client) EmailBreaches(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/email/breaches", map[string]string{"email": email})
}

// EmailVerify performs a deep SMTP verification of an email address.
// POST /api/email/verify
func (c *Client) EmailVerify(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/email/verify", map[string]string{"email": email})
}

// EmailValidityBulk validates a batch of emails in a single synchronous request
// (payload up to 64MB).
// POST /api/email/validity/bulk
func (c *Client) EmailValidityBulk(ctx context.Context, emails []string) (json.RawMessage, error) {
	return c.post(ctx, "/api/email/validity/bulk", map[string][]string{"emails": emails})
}
