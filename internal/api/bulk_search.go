package api

import (
	"context"
	"encoding/json"
)

// BulkEvent is a single server-sent event emitted by a bulk search stream.
// Type is one of: start, result, error, done.
type BulkEvent struct {
	Type string
	Data json.RawMessage
}

// BulkValiditySearch streams validity results for many emails over SSE.
// POST /api/cli/bulk-validity-search
func (c *Client) BulkValiditySearch(ctx context.Context, queries []string, fileName string, onEvent func(BulkEvent) error) error {
	return c.bulkSearch(ctx, "/api/cli/bulk-validity-search", queries, fileName, onEvent)
}

// BulkBreachesSearch streams breach results for many emails over SSE.
// POST /api/cli/bulk-breaches-search
func (c *Client) BulkBreachesSearch(ctx context.Context, queries []string, fileName string, onEvent func(BulkEvent) error) error {
	return c.bulkSearch(ctx, "/api/cli/bulk-breaches-search", queries, fileName, onEvent)
}

func (c *Client) bulkSearch(ctx context.Context, path string, queries []string, fileName string, onEvent func(BulkEvent) error) error {
	payload := map[string]interface{}{"queries": queries}
	if fileName != "" {
		payload["file_name"] = fileName
	}
	return c.stream(ctx, path, payload, func(eventType string, data json.RawMessage) error {
		if eventType == "" {
			eventType = "result"
		}
		return onEvent(BulkEvent{Type: eventType, Data: data})
	})
}
