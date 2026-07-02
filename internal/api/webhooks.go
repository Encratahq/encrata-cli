package api

import (
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

func (c *Client) ListWebhooks() (json.RawMessage, error) {
	return c.get("/api/webhooks", nil)
}

func (c *Client) CreateWebhook(req *WebhookRequest) (json.RawMessage, error) {
	return c.post("/api/webhooks", req)
}

func (c *Client) UpdateWebhook(req *WebhookRequest) (json.RawMessage, error) {
	return c.put("/api/webhooks", req)
}

func (c *Client) DeleteWebhook(id string) (json.RawMessage, error) {
	return c.del("/api/webhooks", nil, map[string]string{"id": id})
}

func (c *Client) TestWebhook(id string) (json.RawMessage, error) {
	return c.post("/api/webhooks/test", map[string]string{"id": id})
}

func (c *Client) ListWebhookDeliveries(id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("webhook_id", id)
	return c.get("/api/webhooks/deliveries", q)
}
