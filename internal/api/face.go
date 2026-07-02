package api

import "encoding/json"

type FaceRequest struct {
	ImageURL  string   `json:"image_url"`
	Threshold *float64 `json:"threshold,omitempty"`
}

func (c *Client) FaceSearch(req *FaceRequest) (json.RawMessage, error) {
	return c.post("/api/agent/face", req)
}
