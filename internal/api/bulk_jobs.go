package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *Client) CreateBulkJob(ctx context.Context, emails []string) (json.RawMessage, error) {
	return c.post(ctx, "/api/bulk-jobs", map[string][]string{"emails": emails})
}

func (c *Client) ListBulkJobs(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/bulk-jobs", nil)
}

func (c *Client) GetBulkJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.get(ctx, "/api/bulk-jobs", q)
}

func (c *Client) CancelBulkJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.del(ctx, "/api/bulk-jobs", q, nil)
}

func (c *Client) DownloadBulkJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.get(ctx, "/api/bulk-jobs/download", q)
}

func BulkJobID(data json.RawMessage) string {
	var resp struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(data, &resp) == nil {
		return resp.ID
	}
	return ""
}

func BulkJobDownloadURL(data json.RawMessage) string {
	var resp struct {
		DownloadURL string `json:"download_url"`
		Job         struct {
			OutputKey string `json:"output_key"`
		} `json:"job"`
	}
	if json.Unmarshal(data, &resp) == nil {
		return resp.DownloadURL
	}
	return ""
}

func BulkJobError(action string, err error) error {
	return fmt.Errorf("bulk job %s failed: %w", action, err)
}
