package api

import (
	"context"
	"encoding/json"
	"net/url"
)

type WebhookRequest struct {
	ID          string   `json:"id,omitempty"`
	URL         string   `json:"url,omitempty"`
	Events      []string `json:"events,omitempty"`
	Description string   `json:"description,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

func (c *Client) ListWebhooks(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/webhooks", nil)
}

func (c *Client) CreateWebhook(ctx context.Context, req *WebhookRequest) (json.RawMessage, error) {
	return c.post(ctx, "/api/webhooks", req)
}

func (c *Client) UpdateWebhook(ctx context.Context, req *WebhookRequest) (json.RawMessage, error) {
	return c.put(ctx, "/api/webhooks", req)
}

func (c *Client) DeleteWebhook(ctx context.Context, id string) (json.RawMessage, error) {
	return c.del(ctx, "/api/webhooks", nil, map[string]string{"id": id})
}

func (c *Client) TestWebhook(ctx context.Context, id string) (json.RawMessage, error) {
	return c.post(ctx, "/api/webhooks/test", map[string]string{"id": id})
}

func (c *Client) ListWebhookDeliveries(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("webhook_id", id)
	return c.get(ctx, "/api/webhooks/deliveries", q)
}
