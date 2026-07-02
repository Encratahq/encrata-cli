package api

import (
	"encoding/json"
	"net/url"
	"strconv"
)

type WorkflowCreateRequest struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	TemplateID  string                   `json:"template_id,omitempty"`
	Trigger     map[string]interface{}   `json:"trigger,omitempty"`
	Steps       []map[string]interface{} `json:"steps,omitempty"`
}

type WorkflowUpdateRequest struct {
	Name        string                   `json:"name,omitempty"`
	Description string                   `json:"description,omitempty"`
	Status      string                   `json:"status,omitempty"`
	Trigger     map[string]interface{}   `json:"trigger,omitempty"`
	Steps       []map[string]interface{} `json:"steps,omitempty"`
}

func (c *Client) ListWorkflows(page, limit int, status string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	if status != "" {
		q.Set("status", status)
	}
	return c.get("/api/workflows", q)
}

func (c *Client) CreateWorkflow(req *WorkflowCreateRequest) (json.RawMessage, error) {
	return c.post("/api/workflows", req)
}

func (c *Client) UpdateWorkflow(id string, req *WorkflowUpdateRequest) (json.RawMessage, error) {
	return c.put("/api/workflows/"+url.PathEscape(id), req)
}

func (c *Client) GetWorkflow(id string) (json.RawMessage, error) {
	return c.get("/api/workflows/"+url.PathEscape(id), nil)
}

func (c *Client) ListWorkflowRuns(page, limit int, workflowID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	if workflowID != "" {
		q.Set("workflow_id", workflowID)
	}
	return c.get("/api/workflows/runs", q)
}

func (c *Client) GetWorkflowRun(id string) (json.RawMessage, error) {
	return c.get("/api/workflows/runs/"+url.PathEscape(id), nil)
}

func (c *Client) ListWorkflowTemplates(category string) (json.RawMessage, error) {
	q := url.Values{}
	if category != "" {
		q.Set("category", category)
	}
	return c.get("/api/workflows/templates", q)
}

func (c *Client) ListWorkflowSecrets() (json.RawMessage, error) {
	return c.get("/api/workflows/secrets", nil)
}

func (c *Client) CreateWorkflowSecret(name, value string) (json.RawMessage, error) {
	return c.post("/api/workflows/secrets", map[string]string{"name": name, "value": value})
}

func (c *Client) DeleteWorkflowSecret(name string) (json.RawMessage, error) {
	return c.del("/api/workflows/secrets", nil, map[string]string{"name": name})
}
