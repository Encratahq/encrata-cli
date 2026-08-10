package api

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// Workflow bulk-enrichment + integrations API. All routes live on the API-key
// agent surface (/api/agent/workflows*), self-authenticated by the backend with
// the CLI's configured enc_ key. Response shapes mirror the workflows service
// handlers (workflows.go and workflow_integrations.go).

// UploadWorkflowFile uploads a CSV/TXT/XLSX file for bulk enrichment.
// POST /api/agent/workflows/files (multipart; field "file", optional "workflow_id").
// Response: {id, filename, row_count, identifier_type, identifier_column}.
func (c *Client) UploadWorkflowFile(ctx context.Context, fileName string, data []byte, workflowID string) (json.RawMessage, error) {
	fields := map[string]string{}
	if workflowID != "" {
		fields["workflow_id"] = workflowID
	}
	return c.postMultipart(ctx, "/api/agent/workflows/files", "file", fileName, data, fields)
}

// RunWorkflow triggers a bulk run over an uploaded file.
// POST /api/workflows/{id}/run with body {file_id}. The backend stores the body
// as trigger_data and the worker reads file_id/file_ids. An Idempotency-Key
// header (when supplied) lets the caller retry safely without duplicate runs.
// Response: the full WorkflowRun object (201).
func (c *Client) RunWorkflow(ctx context.Context, workflowID, fileID, idempotencyKey string) (json.RawMessage, error) {
	path := "/api/agent/workflows/" + url.PathEscape(workflowID) + "/run"
	body := map[string]interface{}{}
	if fileID != "" {
		body["file_id"] = fileID
	}
	headers := map[string]string{}
	if idempotencyKey != "" {
		headers["Idempotency-Key"] = idempotencyKey
	}
	return c.postWithHeaders(ctx, path, body, headers)
}

// GetWorkflowRun returns a run's status and steps.
// GET /api/agent/workflows/runs/{id} -> {run, steps}.
func (c *Client) GetWorkflowRun(ctx context.Context, runID string) (json.RawMessage, error) {
	return c.get(ctx, "/api/agent/workflows/runs/"+url.PathEscape(runID), nil)
}

// ListWorkflowRuns lists recent runs, optionally scoped to one workflow.
// GET /api/workflows/runs -> {runs, total, limit, offset}.
func (c *Client) ListWorkflowRuns(ctx context.Context, workflowID string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	if workflowID != "" {
		q.Set("workflow_id", workflowID)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return c.get(ctx, "/api/agent/workflows/runs", q)
}

// CancelWorkflowRun requests cooperative cancellation of a run.
// POST /api/agent/workflows/runs/{id}/cancel -> {ok, cancel_requested}.
func (c *Client) CancelWorkflowRun(ctx context.Context, runID string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/workflows/runs/"+url.PathEscape(runID)+"/cancel", nil)
}

// DownloadWorkflowRunOutput follows the 302 redirect to the presigned CSV and
// returns the raw bytes. GET /api/agent/workflows/runs/{id}/download.
// The Go HTTP client follows the redirect automatically and drops the
// Authorization header cross-host, which is correct — the S3 URL is presigned.
func (c *Client) DownloadWorkflowRunOutput(ctx context.Context, runID string) ([]byte, error) {
	return c.getBytes(ctx, "/api/agent/workflows/runs/"+url.PathEscape(runID)+"/download", nil)
}

// ── Integrations ────────────────────────────────────────────────────────────

// ListIntegrations lists connected accounts (never includes tokens/secrets).
// GET /api/agent/workflows/integrations -> {integrations}.
func (c *Client) ListIntegrations(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/agent/workflows/integrations", nil)
}

// NangoProviders lists connectable providers plus the Nango host and connect URL.
// GET /api/agent/workflows/integrations/nango/providers -> {providers, host, connect_url}.
// Returns a 503 api.Error when NANGO_SECRET_KEY is not configured server-side.
func (c *Client) NangoProviders(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/agent/workflows/integrations/nango/providers", nil)
}

// NangoSession starts a Nango Connect session and returns a session token.
// POST /api/agent/workflows/integrations/nango/session -> {token, expires_at, host, connect_url}.
func (c *Client) NangoSession(ctx context.Context, integration string) (json.RawMessage, error) {
	body := map[string]interface{}{}
	if integration != "" {
		body["integration"] = integration
	}
	return c.post(ctx, "/api/agent/workflows/integrations/nango/session", body)
}

// NangoSave persists a completed Nango connection.
// POST /api/agent/workflows/integrations/nango/save -> the saved integration (201).
func (c *Client) NangoSave(ctx context.Context, connectionID, providerConfigKey, label string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/workflows/integrations/nango/save", map[string]interface{}{
		"connection_id":       connectionID,
		"provider_config_key": providerConfigKey,
		"label":               label,
	})
}

// CreateIntegrationSheet creates a spreadsheet via the Nango proxy for a
// connected Google account. The live route suffix is /sheet (not /create-sheet).
// POST /api/agent/workflows/integrations/{id}/sheet -> {spreadsheet_id, spreadsheet_url, sheet_name}.
func (c *Client) CreateIntegrationSheet(ctx context.Context, id, title string) (json.RawMessage, error) {
	body := map[string]interface{}{}
	if title != "" {
		body["title"] = title
	}
	return c.post(ctx, "/api/agent/workflows/integrations/"+url.PathEscape(id)+"/sheet", body)
}

// DeleteIntegration disconnects a connected account.
// DELETE /api/agent/workflows/integrations/{id} -> {deleted}.
func (c *Client) DeleteIntegration(ctx context.Context, id string) (json.RawMessage, error) {
	return c.del(ctx, "/api/agent/workflows/integrations/"+url.PathEscape(id), nil, nil)
}
