package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ExportState is the async cached-export status returned by /api/cli/jobs/exports*.
// Status is one of: none, building, ready, failed.
type ExportState struct {
	Status   string `json:"status"`
	URL      string `json:"url"`
	RowCount int    `json:"row_count"`
	Error    string `json:"error"`
}

func exportQuery(kind, id, format, filter string) url.Values {
	q := url.Values{}
	q.Set("kind", kind)
	q.Set("id", id)
	q.Set("format", format)
	if filter == "" {
		filter = "all"
	}
	q.Set("filter", filter)
	return q
}

// CreateExport enqueues (or reuses) a cached export build for a completed job.
// A ready variant returns its presigned URL immediately; otherwise the server
// starts building and returns status "building". POST /api/cli/jobs/exports
func (c *Client) CreateExport(ctx context.Context, kind, id, format, filter string) (*ExportState, error) {
	if filter == "" {
		filter = "all"
	}
	data, err := c.post(ctx, "/api/cli/jobs/exports", map[string]string{
		"kind": kind, "id": id, "format": format, "filter": filter,
	})
	if err != nil {
		return nil, err
	}
	return decodeExportState(data)
}

// ExportStatus polls a variant's build state. GET /api/cli/jobs/exports/status
func (c *Client) ExportStatus(ctx context.Context, kind, id, format, filter string) (*ExportState, error) {
	data, err := c.get(ctx, "/api/cli/jobs/exports/status", exportQuery(kind, id, format, filter))
	if err != nil {
		return nil, err
	}
	return decodeExportState(data)
}

// ExportDownloadURL resolves a fresh presigned attachment URL for a ready
// variant, transparently rebuilding if the object aged out (status "building").
// GET /api/cli/jobs/exports/download?json=1
func (c *Client) ExportDownloadURL(ctx context.Context, kind, id, format, filter string) (*ExportState, error) {
	q := exportQuery(kind, id, format, filter)
	q.Set("json", "1")
	data, err := c.get(ctx, "/api/cli/jobs/exports/download", q)
	if err != nil {
		return nil, err
	}
	return decodeExportState(data)
}

// FetchURL streams the bytes of a presigned S3 URL. It sends no API credentials,
// since the URL is already signed; the file is pulled straight from storage.
func (c *Client) FetchURL(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	return data, nil
}

func decodeExportState(data json.RawMessage) (*ExportState, error) {
	var s ExportState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to decode export status: %w", err)
	}
	if s.Status == "" {
		// A bare {url} (already-built variant) still counts as ready.
		if s.URL != "" {
			s.Status = "ready"
		} else {
			s.Status = "building"
		}
	}
	return &s, nil
}
