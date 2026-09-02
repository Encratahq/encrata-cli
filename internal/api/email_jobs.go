package api

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// CreateIdentityJob creates an async identity enrichment job.
// POST /api/cli/jobs/identity
func (c *Client) CreateIdentityJob(ctx context.Context, emails []string, fileName string) (json.RawMessage, error) {
	payload := map[string]interface{}{"emails": emails}
	if fileName != "" {
		payload["file_name"] = fileName
	}
	return c.post(ctx, "/api/cli/jobs/identity", payload)
}

// CreatePasswordJob creates an async password breach job from SHA-1 hashes.
// POST /api/cli/jobs/password
func (c *Client) CreatePasswordJob(ctx context.Context, sha1s []string, fileName string) (json.RawMessage, error) {
	payload := map[string]interface{}{"sha1s": sha1s}
	if fileName != "" {
		payload["file_name"] = fileName
	}
	return c.post(ctx, "/api/cli/jobs/password", payload)
}

// GetIdentityJob fetches an identity job by id.
// GET /api/cli/jobs/identity?id=
func (c *Client) GetIdentityJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.get(ctx, "/api/cli/jobs/identity", q)
}

// ListIdentityJobs lists identity jobs (paginated).
// GET /api/cli/jobs/identity
func (c *Client) ListIdentityJobs(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/cli/jobs/identity", nil)
}

// GetPasswordJob fetches a password job by id.
// GET /api/cli/jobs/password?id=
func (c *Client) GetPasswordJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.get(ctx, "/api/cli/jobs/password", q)
}

// ListPasswordJobs lists password jobs (paginated).
// GET /api/cli/jobs/password
func (c *Client) ListPasswordJobs(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/cli/jobs/password", nil)
}

// GetIdentityJobResults fetches paginated identity job rows.
// GET /api/cli/jobs/identity/results?id=&page=&page_size=&found=1
func (c *Client) GetIdentityJobResults(ctx context.Context, id string, page, pageSize int, foundOnly bool) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	if foundOnly {
		q.Set("found", "1")
	}
	return c.get(ctx, "/api/cli/jobs/identity/results", q)
}

// GetValidityJobResultsPage fetches paginated validity job rows.
// GET /api/cli/jobs/validity/results?id=&page=&page_size=
func (c *Client) GetValidityJobResultsPage(ctx context.Context, id string, page, pageSize int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	return c.get(ctx, "/api/cli/jobs/validity/results", q)
}

// GetPasswordJobResults fetches paginated password job rows.
// GET /api/cli/jobs/password/results?id=&page=&page_size=&breached=1
func (c *Client) GetPasswordJobResults(ctx context.Context, id string, page, pageSize int, breachedOnly bool) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	if breachedOnly {
		q.Set("breached", "1")
	}
	return c.get(ctx, "/api/cli/jobs/password/results", q)
}

// DownloadIdentityJob downloads identity job output bytes.
// GET /api/cli/jobs/identity/download?id=&found=1
func (c *Client) DownloadIdentityJob(ctx context.Context, id string, foundOnly bool) ([]byte, error) {
	q := url.Values{}
	q.Set("id", id)
	if foundOnly {
		q.Set("found", "1")
	}
	return c.getBytes(ctx, "/api/cli/jobs/identity/download", q)
}

// DownloadPasswordJob downloads password job output bytes.
// GET /api/cli/jobs/password/download?id=&breached=1
func (c *Client) DownloadPasswordJob(ctx context.Context, id string, breachedOnly bool) ([]byte, error) {
	q := url.Values{}
	q.Set("id", id)
	if breachedOnly {
		q.Set("breached", "1")
	}
	return c.getBytes(ctx, "/api/cli/jobs/password/download", q)
}

// CancelIdentityJob cancels a running identity job.
// POST /api/cli/jobs/identity/cancel?id=
func (c *Client) CancelIdentityJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.postQuery(ctx, "/api/cli/jobs/identity/cancel", q, nil)
}

// CancelPasswordJob cancels a running password job.
// POST /api/cli/jobs/password/cancel?id=
func (c *Client) CancelPasswordJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.postQuery(ctx, "/api/cli/jobs/password/cancel", q, nil)
}

// RetryValidityJob retries dead-lettered validity chunks.
// POST /api/cli/jobs/validity/retry?id=
func (c *Client) RetryValidityJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.postQuery(ctx, "/api/cli/jobs/validity/retry", q, nil)
}

// RetryIdentityJob retries dead-lettered identity chunks.
// POST /api/cli/jobs/identity/retry?id=
func (c *Client) RetryIdentityJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.postQuery(ctx, "/api/cli/jobs/identity/retry", q, nil)
}

// RetryPasswordJob retries dead-lettered password chunks.
// POST /api/cli/jobs/password/retry?id=
func (c *Client) RetryPasswordJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.postQuery(ctx, "/api/cli/jobs/password/retry", q, nil)
}

// ListBulkJobs lists all async bulk jobs.
// GET /api/cli/jobs/bulk
func (c *Client) ListBulkJobs(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/cli/jobs/bulk", nil)
}

// GetBulkJob fetches one bulk job by id.
// GET /api/cli/jobs/bulk?id=
func (c *Client) GetBulkJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.get(ctx, "/api/cli/jobs/bulk", q)
}

// CancelBulkJob cancels a pending/in-progress bulk job.
// DELETE /api/cli/jobs/bulk?id=
func (c *Client) CancelBulkJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.del(ctx, "/api/cli/jobs/bulk", q, nil)
}
