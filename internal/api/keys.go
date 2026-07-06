package api

import (
	"context"
	"encoding/json"
	"net/url"
)

func (c *Client) ListKeys(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/keys", nil)
}

func (c *Client) CreateKey(ctx context.Context, name string) (json.RawMessage, error) {
	return c.post(ctx, "/api/keys", map[string]string{"name": name})
}

func (c *Client) RevokeKey(ctx context.Context, id string, permanent bool) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	if permanent {
		q.Set("permanent", "true")
	}
	return c.del(ctx, "/api/keys", q, nil)
}
