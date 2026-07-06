package api

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

type MonitorCreateRequest struct {
	Name            string   `json:"name"`
	Frequency       string   `json:"frequency"`
	ChangeDetection string   `json:"change_detection"`
	Emails          []string `json:"emails,omitempty"`
	DataSourceType  string   `json:"data_source_type,omitempty"`
	DataSourceRef   string   `json:"data_source_ref,omitempty"`
}

func (c *Client) ListMonitors(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/agent/monitors", nil)
}

func (c *Client) CreateMonitor(ctx context.Context, req *MonitorCreateRequest) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/monitors", req)
}

func (c *Client) GetMonitor(ctx context.Context, id string) (json.RawMessage, error) {
	return c.get(ctx, "/api/agent/monitors/"+url.PathEscape(id), nil)
}

func (c *Client) TriggerMonitorRun(ctx context.Context, id string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/monitors/"+url.PathEscape(id)+"/run", map[string]string{})
}

func (c *Client) ListMonitorRuns(ctx context.Context, id string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	return c.get(ctx, "/api/agent/monitors/"+url.PathEscape(id)+"/runs", q)
}

func (c *Client) GetMonitorRunResults(ctx context.Context, monitorID, runID string, changesOnly bool, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	if changesOnly {
		q.Set("changes_only", "true")
	}
	path := "/api/agent/monitors/" + url.PathEscape(monitorID) + "/runs/" + url.PathEscape(runID) + "/results"
	return c.get(ctx, path, q)
}

func (c *Client) ListAllMonitorRuns(ctx context.Context, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	return c.get(ctx, "/api/agent/monitoring/runs", q)
}

func (c *Client) ListAllMonitorResults(ctx context.Context, changesOnly bool, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	if changesOnly {
		q.Set("changes_only", "true")
	}
	return c.get(ctx, "/api/agent/monitoring/results", q)
}
