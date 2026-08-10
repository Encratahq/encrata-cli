package api

import (
	"context"
	"encoding/json"
)

// PasswordBreaches checks a single password against known data breaches.
//
// Only the UPPER-CASE hex SHA-1 hash is transmitted (field "sha1"); the
// plaintext password is never sent, logged, or stored.
// POST /api/agent/password-breaches
func (c *Client) PasswordBreaches(ctx context.Context, sha1 string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/password-breaches", map[string]string{"sha1": sha1})
}

// PasswordBreachesBulk checks many unique SHA-1 hashes (max 1000) in one call.
//
// Only UPPER-CASE hex SHA-1 hashes are transmitted (field "sha1s"); plaintext
// passwords are never sent.
// POST /api/agent/password-breaches/bulk
func (c *Client) PasswordBreachesBulk(ctx context.Context, sha1s []string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/password-breaches/bulk", map[string][]string{"sha1s": sha1s})
}
