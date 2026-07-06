package api

import (
	"context"
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

func (c *Client) ListWorkflows(ctx context.Context, page, limit int, status string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	if status != "" {
		q.Set("status", status)
	}
	return c.get(ctx, "/api/workflows", q)
}

func (c *Client) CreateWorkflow(ctx context.Context, req *WorkflowCreateRequest) (json.RawMessage, error) {
	return c.post(ctx, "/api/workflows", req)
}

func (c *Client) UpdateWorkflow(ctx context.Context, id string, req *WorkflowUpdateRequest) (json.RawMessage, error) {
	return c.put(ctx, "/api/workflows/"+url.PathEscape(id), req)
}

func (c *Client) GetWorkflow(ctx context.Context, id string) (json.RawMessage, error) {
	return c.get(ctx, "/api/workflows/"+url.PathEscape(id), nil)
}

func (c *Client) ListWorkflowRuns(ctx context.Context, page, limit int, workflowID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	if workflowID != "" {
		q.Set("workflow_id", workflowID)
	}
	return c.get(ctx, "/api/workflows/runs", q)
}

func (c *Client) GetWorkflowRun(ctx context.Context, id string) (json.RawMessage, error) {
	return c.get(ctx, "/api/workflows/runs/"+url.PathEscape(id), nil)
}

func (c *Client) ListWorkflowTemplates(ctx context.Context, category string) (json.RawMessage, error) {
	q := url.Values{}
	if category != "" {
		q.Set("category", category)
	}
	return c.get(ctx, "/api/workflows/templates", q)
}

func (c *Client) ListWorkflowSecrets(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/workflows/secrets", nil)
}

func (c *Client) CreateWorkflowSecret(ctx context.Context, name, value string) (json.RawMessage, error) {
	return c.post(ctx, "/api/workflows/secrets", map[string]string{"name": name, "value": value})
}

func (c *Client) DeleteWorkflowSecret(ctx context.Context, name string) (json.RawMessage, error) {
	return c.del(ctx, "/api/workflows/secrets", nil, map[string]string{"name": name})
}
