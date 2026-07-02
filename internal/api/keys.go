package api

import (
	"encoding/json"
	"net/url"
)

func (c *Client) ListKeys() (json.RawMessage, error) {
	return c.get("/api/keys", nil)
}

func (c *Client) CreateKey(name string) (json.RawMessage, error) {
	return c.post("/api/keys", map[string]string{"name": name})
}

func (c *Client) RevokeKey(id string, permanent bool) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	if permanent {
		q.Set("permanent", "true")
	}
	return c.del("/api/keys", q, nil)
}
