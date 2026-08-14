package api

import (
	"context"
	"encoding/json"
	"net/url"
)

func (c *Client) ListKeys(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/cli/keys", nil)
}

func (c *Client) CreateKey(ctx context.Context, name string) (json.RawMessage, error) {
	return c.post(ctx, "/api/cli/keys", map[string]string{"name": name})
}

func (c *Client) RevokeKey(ctx context.Context, id string, permanent bool) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	if permanent {
		q.Set("permanent", "true")
	}
	return c.del(ctx, "/api/cli/keys", q, nil)
}

// RenameKey renames an API key.
// PUT /api/cli/keys  { "id": <id>, "name": <name> }
func (c *Client) RenameKey(ctx context.Context, id, name string) (json.RawMessage, error) {
	return c.put(ctx, "/api/cli/keys", map[string]string{"id": id, "name": name})
}

// SetKeyStatus enables or disables an API key without deleting it.
// PATCH /api/cli/keys?id=<id>&action=enable|disable
func (c *Client) SetKeyStatus(ctx context.Context, id, action string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	q.Set("action", action)
	return c.patchQuery(ctx, "/api/cli/keys", q, nil)
}

// SetKeyCreditLimit sets or clears an API key's credit limit. A nil limit
// clears the limit (unlimited).
// POST /api/cli/keys/limit  { "id": <id>, "credit_limit": <N|null> }
func (c *Client) SetKeyCreditLimit(ctx context.Context, id string, limit *int) (json.RawMessage, error) {
	return c.post(ctx, "/api/cli/keys/limit", map[string]interface{}{"id": id, "credit_limit": limit})
}
