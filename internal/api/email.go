package api

import (
	"context"
	"encoding/json"
)

// EmailValidity checks whether a single email address is valid and deliverable.
// POST /api/agent/email-validity
func (c *Client) EmailValidity(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/email-validity", map[string]string{"email": email})
}

// EmailEnrich returns validity plus enrichment (person/company) data.
// POST /api/agent/email-enrich
func (c *Client) EmailEnrich(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/email-enrich", map[string]string{"email": email})
}

// EmailIdentity resolves the identity behind an email address.
// POST /api/agent/email-identity
func (c *Client) EmailIdentity(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/email-identity", map[string]string{"email": email})
}

// EmailBreaches lists known data breaches an email appears in.
// POST /api/agent/breaches
func (c *Client) EmailBreaches(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/breaches", map[string]string{"email": email})
}

// EmailVerify performs a deep SMTP verification of an email address.
// POST /api/agent/email-verify
func (c *Client) EmailVerify(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/email-verify", map[string]string{"email": email})
}

// EmailValidityBulk validates a batch of emails in a single synchronous request
// (payload up to 64MB).
// POST /api/agent/email-validity-bulk
func (c *Client) EmailValidityBulk(ctx context.Context, emails []string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/email-validity-bulk", map[string][]string{"emails": emails})
}
