package api

import (
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

func (c *Client) ListMonitors() (json.RawMessage, error) {
	return c.get("/api/agent/monitors", nil)
}

func (c *Client) CreateMonitor(req *MonitorCreateRequest) (json.RawMessage, error) {
	return c.post("/api/agent/monitors", req)
}

func (c *Client) GetMonitor(id string) (json.RawMessage, error) {
	return c.get("/api/agent/monitors/"+url.PathEscape(id), nil)
}

func (c *Client) TriggerMonitorRun(id string) (json.RawMessage, error) {
	return c.post("/api/agent/monitors/"+url.PathEscape(id)+"/run", map[string]string{})
}

func (c *Client) ListMonitorRuns(id string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	return c.get("/api/agent/monitors/"+url.PathEscape(id)+"/runs", q)
}

func (c *Client) GetMonitorRunResults(monitorID, runID string, changesOnly bool, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	if changesOnly {
		q.Set("changes_only", "true")
	}
	path := "/api/agent/monitors/" + url.PathEscape(monitorID) + "/runs/" + url.PathEscape(runID) + "/results"
	return c.get(path, q)
}

func (c *Client) ListAllMonitorRuns(limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	return c.get("/api/agent/monitoring/runs", q)
}

func (c *Client) ListAllMonitorResults(changesOnly bool, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	if changesOnly {
		q.Set("changes_only", "true")
	}
	return c.get("/api/agent/monitoring/results", q)
}
