package api

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
)

type EmailRequest struct {
	Email   string   `json:"email"`
	Country string   `json:"country,omitempty"`
	Lang    string   `json:"lang,omitempty"`
	Num     int      `json:"num,omitempty"`
	Page    int      `json:"page,omitempty"`
	Fields  []string `json:"-"`
}

func (c *Client) EmailLookup(ctx context.Context, req *EmailRequest, nocache bool) (json.RawMessage, error) {
	q := url.Values{}
	if len(req.Fields) > 0 {
		q.Set("fields", strings.Join(req.Fields, ","))
	}
	if nocache {
		q.Set("nocache", "1")
	}
	return c.postQuery(ctx, "/api/agent/lookup", q, req)
}

func (c *Client) Validate(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/validate", map[string]string{"email": email})
}

func (c *Client) Breaches(ctx context.Context, email string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/breaches", map[string]string{"email": email})
}
