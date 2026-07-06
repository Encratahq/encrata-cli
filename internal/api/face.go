package api

import (
	"context"
	"encoding/json"
)

type FaceRequest struct {
	ImageURL  string   `json:"image_url"`
	Threshold *float64 `json:"threshold,omitempty"`
}

func (c *Client) FaceSearch(ctx context.Context, req *FaceRequest) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/face", req)
}
