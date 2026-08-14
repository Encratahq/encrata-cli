package api

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Job is the status view of an async validity job.
type Job struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	FileName       string `json:"file_name,omitempty"`
	BatchID        string `json:"batch_id,omitempty"`
	TotalEmails    int    `json:"total_emails"`
	ProcessedCount int    `json:"processed_count"`
	SuccessCount   int    `json:"success_count"`
	ErrorCount     int    `json:"error_count"`
	CreditsUsed    int    `json:"credits_used"`
	CreatedAt      string `json:"created_at,omitempty"`
}

// CreateJobRequest is the JSON body for creating a validity job.
type CreateJobRequest struct {
	Emails   []string `json:"emails,omitempty"`
	FileName string   `json:"file_name,omitempty"`
	BatchID  string   `json:"batch_id,omitempty"`
}

// CreateValidityJob creates an async validity job from a list of emails.
// POST /api/cli/validity-jobs
func (c *Client) CreateValidityJob(ctx context.Context, req *CreateJobRequest) (json.RawMessage, error) {
	return c.post(ctx, "/api/cli/validity-jobs", req)
}

// CreateValidityJobFile creates an async validity job from an uploaded file
// (multipart, up to 50MB).
// POST /api/cli/validity-jobs
func (c *Client) CreateValidityJobFile(ctx context.Context, fileName string, data []byte) (json.RawMessage, error) {
	return c.postMultipart(ctx, "/api/cli/validity-jobs", "file", fileName, data, nil)
}

// GetValidityJob fetches a single job by id.
// GET /api/cli/validity-jobs?id=
func (c *Client) GetValidityJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.get(ctx, "/api/cli/validity-jobs", q)
}

// ListValidityJobs lists all jobs.
// GET /api/cli/validity-jobs
func (c *Client) ListValidityJobs(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/cli/validity-jobs", nil)
}

// GetValidityJobResults returns a page of job results, optionally filtered by
// per-row status.
// GET /api/cli/validity-jobs/results?id=&page=&status=
func (c *Client) GetValidityJobResults(ctx context.Context, id string, page int, status string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if status != "" {
		q.Set("status", status)
	}
	return c.get(ctx, "/api/cli/validity-jobs/results", q)
}

// DownloadValidityJob downloads job results as raw CSV or JSON bytes.
// GET /api/cli/validity-jobs/download?id=&status=&format=
func (c *Client) DownloadValidityJob(ctx context.Context, id, status, format string) ([]byte, error) {
	q := url.Values{}
	q.Set("id", id)
	if status != "" {
		q.Set("status", status)
	}
	if format != "" {
		q.Set("format", format)
	}
	return c.getBytes(ctx, "/api/cli/validity-jobs/download", q)
}

// CancelValidityJob cancels a running job.
// POST /api/cli/validity-jobs/cancel?id=
func (c *Client) CancelValidityJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.postQuery(ctx, "/api/cli/validity-jobs/cancel", q, nil)
}

// ParseJob decodes a job from an API response, whether it is bare or nested
// under a "job" key.
func ParseJob(data json.RawMessage) (*Job, error) {
	var wrap struct {
		Job *Job `json:"job"`
	}
	if json.Unmarshal(data, &wrap) == nil && wrap.Job != nil {
		return wrap.Job, nil
	}
	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// JobIsTerminal reports whether a status represents a finished job.
func JobIsTerminal(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "complete", "done", "finished", "failed", "error", "cancelled", "canceled":
		return true
	}
	return false
}

// PollValidityJob polls a job until it reaches a terminal state, calling
// onUpdate after each poll.
func (c *Client) PollValidityJob(ctx context.Context, id string, interval time.Duration, onUpdate func(*Job)) (*Job, error) {
	for {
		data, err := c.GetValidityJob(ctx, id)
		if err != nil {
			return nil, err
		}
		job, err := ParseJob(data)
		if err != nil {
			return nil, err
		}
		if onUpdate != nil {
			onUpdate(job)
		}
		if JobIsTerminal(job.Status) {
			return job, nil
		}
		if err := sleepCtx(ctx, interval); err != nil {
			return nil, err
		}
	}
}
