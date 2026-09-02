package api

import (
	"context"
	"encoding/json"
)

// EmailValidity checks whether a single email address is valid and deliverable.
// POST /api/cli/email/validity
func (c *Client) EmailValidity(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/cli/email/validity", map[string]string{"email": email})
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

// EmailValidityBulk validates a batch of emails in a single synchronous request
// (payload up to 64MB).
// POST /api/cli/email/validity/bulk
func (c *Client) EmailValidityBulk(ctx context.Context, emails []string) (json.RawMessage, error) {
	return c.post(ctx, "/api/cli/email/validity/bulk", map[string][]string{"emails": emails})
}
