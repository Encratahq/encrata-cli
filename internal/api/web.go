package api

import (
	"context"
	"encoding/json"
)

type ScrapeRequest struct {
	URL           string `json:"url"`
	RenderJS      *bool  `json:"render_js,omitempty"`
	BlockAds      *bool  `json:"block_ads,omitempty"`
	BlockTrackers *bool  `json:"block_trackers,omitempty"`
	WaitFor       string `json:"wait_for,omitempty"`
	Timeout       int    `json:"timeout,omitempty"`
}

func (c *Client) Scrape(ctx context.Context, req *ScrapeRequest) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/scrape", req)
}

type ExtractRequest struct {
	URL           string            `json:"url"`
	Mode          string            `json:"mode,omitempty"`
	Selectors     map[string]string `json:"selectors,omitempty"`
	RenderJS      *bool             `json:"render_js,omitempty"`
	BlockAds      *bool             `json:"block_ads,omitempty"`
	BlockTrackers *bool             `json:"block_trackers,omitempty"`
	WaitFor       string            `json:"wait_for,omitempty"`
	Timeout       int               `json:"timeout,omitempty"`
}

func (c *Client) Extract(ctx context.Context, req *ExtractRequest) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/extract", req)
}

type ScreenshotRequest struct {
	URL           string `json:"url"`
	FullPage      *bool  `json:"full_page,omitempty"`
	Format        string `json:"format,omitempty"`
	Selector      string `json:"selector,omitempty"`
	RenderJS      *bool  `json:"render_js,omitempty"`
	BlockAds      *bool  `json:"block_ads,omitempty"`
	BlockTrackers *bool  `json:"block_trackers,omitempty"`
	WaitFor       string `json:"wait_for,omitempty"`
	Timeout       int    `json:"timeout,omitempty"`
}

func (c *Client) Screenshot(ctx context.Context, req *ScreenshotRequest) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/screenshot", req)
}
