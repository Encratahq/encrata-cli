package api

import (
	"context"
	"encoding/json"
)

// Me returns the authenticated account: email, name, plan, credits_remaining,
// role, and the active workspace.
// GET /api/cli/me
func (c *Client) Me(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/cli/me", nil)
}
