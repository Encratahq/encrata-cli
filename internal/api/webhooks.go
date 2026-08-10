package api

import (
	"context"
	"encoding/json"
	"net/url"
)

// ListWebhooks returns all webhooks for the current workspace.
// GET /api/agent/webhooks
func (c *Client) ListWebhooks(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/agent/webhooks", nil)
}

// GetWebhook returns a single webhook by id.
// GET /api/agent/webhooks?id=
func (c *Client) GetWebhook(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.get(ctx, "/api/agent/webhooks", q)
}

// CreateWebhook registers a new webhook endpoint. The response includes the
// signing secret, which is only ever returned once (on create).
// POST /api/agent/webhooks
func (c *Client) CreateWebhook(ctx context.Context, targetURL, description string, events []string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/webhooks", map[string]interface{}{
		"url":         targetURL,
		"description": description,
		"events":      events,
	})
}

// UpdateWebhook replaces a webhook's URL, description, events and active state.
// The backend performs a full replace, so all fields must be supplied.
// PUT /api/agent/webhooks
func (c *Client) UpdateWebhook(ctx context.Context, id, targetURL, description string, events []string, isActive *bool) (json.RawMessage, error) {
	return c.put(ctx, "/api/agent/webhooks", map[string]interface{}{
		"id":          id,
		"url":         targetURL,
		"description": description,
		"events":      events,
		"is_active":   isActive,
	})
}

// DeleteWebhook removes a webhook. The id is sent in the JSON body.
// DELETE /api/agent/webhooks
func (c *Client) DeleteWebhook(ctx context.Context, id string) (json.RawMessage, error) {
	return c.del(ctx, "/api/agent/webhooks", nil, map[string]string{"id": id})
}

// TestWebhook delivers a test event to a webhook endpoint and reports the
// result.
// POST /api/agent/webhooks/test
func (c *Client) TestWebhook(ctx context.Context, id string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/webhooks/test", map[string]string{"id": id})
}

// ListWebhookDeliveries returns recent delivery attempts for a webhook.
// GET /api/agent/webhooks/deliveries?webhook_id=
func (c *Client) ListWebhookDeliveries(ctx context.Context, webhookID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("webhook_id", webhookID)
	return c.get(ctx, "/api/agent/webhooks/deliveries", q)
}
