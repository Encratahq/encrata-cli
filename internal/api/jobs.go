package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Validity jobs live on the jobs service's unified surface. The per-kind
// /api/jobs/validity create, results, download, cancel and retry routes were
// retired (create now answers 405, the rest 404): a job is created through the
// typed bulk entrypoint and then addressed by id. Only the per-kind collection
// read (GET /api/jobs/validity[?id=]) is still served, for list and status.
// Every JSON response is wrapped in {success, result, message}.

// Job is the status view of an async validity job.
type Job struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	FileName       string `json:"file_name,omitempty"`
	BatchID        string `json:"batch_id,omitempty"`
	TotalEmails    int    `json:"total_emails"`
	ProcessedCount int    `json:"processed_count"`
	ValidCount     int    `json:"valid_count"`
	InvalidCount   int    `json:"invalid_count"`
	CatchAllCount  int    `json:"catch_all_count"`
	RiskyCount     int    `json:"risky_count"`
	CreditsUsed    int    `json:"credits_used"`
	ErrorMessage   string `json:"error_message,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
}

// CreateJobRequest is the JSON body for creating a validity job.
type CreateJobRequest struct {
	Emails   []string `json:"emails,omitempty"`
	FileName string   `json:"file_name,omitempty"`
	BatchID  string   `json:"batch_id,omitempty"`
}

const (
	validityJobType     = "validity"
	validityPageSize    = 50
	validityMaxPageSize = 200 // the server caps results pages at 200 rows
)

// CreateValidityJob creates an async validity job from a list of emails. Like the
// breaches create it skips the transport's automatic retry: a retried create
// after a lost response would start (and bill) a second job for the same list.
// POST /api/cli/jobs/bulk {"type":"validity","emails":[...]}
func (c *Client) CreateValidityJob(ctx context.Context, req *CreateJobRequest) (json.RawMessage, error) {
	payload := map[string]interface{}{"type": validityJobType, "emails": req.Emails}
	if req.FileName != "" {
		payload["file_name"] = req.FileName
	}
	if req.BatchID != "" {
		payload["batch_id"] = req.BatchID
	}
	return c.postWithHeaders(ctx, "/api/cli/jobs/bulk", payload, nil)
}

// CreateValidityJobFile creates an async validity job from an uploaded file
// (multipart, up to 50MB).
// POST /api/cli/jobs/bulk  (multipart: type=validity, file)
func (c *Client) CreateValidityJobFile(ctx context.Context, fileName string, data []byte) (json.RawMessage, error) {
	return c.postMultipart(ctx, "/api/cli/jobs/bulk", "file", fileName, data,
		map[string]string{"type": validityJobType})
}

// GetValidityJob fetches a single job by id.
// GET /api/cli/jobs/validity?id=
func (c *Client) GetValidityJob(ctx context.Context, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", id)
	return c.get(ctx, "/api/cli/jobs/validity", q)
}

// ListValidityJobs lists all jobs.
// GET /api/cli/jobs/validity
func (c *Client) ListValidityJobs(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/cli/jobs/validity", nil)
}

// GetValidityJobResults returns one page (50 rows) of a job's results,
// optionally filtered by per-row status (valid | invalid | catch-all | risky).
// See validityResultsPage for the response shape.
// GET /api/cli/jobs/{id}?type=validity&cursor=&page_size=&status=
func (c *Client) GetValidityJobResults(ctx context.Context, id string, page int, status string) (json.RawMessage, error) {
	return c.validityResultsPage(ctx, id, page, validityPageSize, status)
}

// validityResultsPage returns page N of a job's rows as
// {"results": [...], "next_cursor": "..."}. The unified results route is keyset
// paged only, so page N is reached by walking the cursor from the start. Each
// row's stored report (a JSON string) is folded into the row, so a row carries
// every report field (outcome, conclusion, extra, ...) while its own email,
// status (the four-bucket verdict) and reason win.
func (c *Client) validityResultsPage(ctx context.Context, id string, page, pageSize int, status string) (json.RawMessage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = validityPageSize
	}
	if pageSize > validityMaxPageSize {
		pageSize = validityMaxPageSize
	}
	cursor := "0" // the first keyset page
	for n := 1; ; n++ {
		q := url.Values{}
		q.Set("type", validityJobType)
		q.Set("cursor", cursor)
		q.Set("page_size", strconv.Itoa(pageSize))
		if s := strings.TrimSpace(status); s != "" {
			q.Set("status", s)
		}
		data, err := c.get(ctx, jobPath(id, ""), q)
		if err != nil {
			return nil, err
		}
		var p struct {
			Items      []map[string]interface{} `json:"items"`
			NextCursor *string                  `json:"next_cursor"`
		}
		if err := json.Unmarshal(unwrapEnvelope(data), &p); err != nil {
			return nil, fmt.Errorf("failed to decode validity results: %w", err)
		}
		next := ""
		if p.NextCursor != nil {
			next = *p.NextCursor
		}
		if n == page {
			return validityResultsBody(p.Items, next)
		}
		if next == "" || next == cursor || len(p.Items) == 0 {
			return validityResultsBody(nil, "") // past the last page
		}
		cursor = next
	}
}

// validityResultsBody builds the page the results commands read, with each
// row's report folded in (see FlattenValidityRow).
func validityResultsBody(items []map[string]interface{}, next string) (json.RawMessage, error) {
	rows := make([]map[string]interface{}, 0, len(items))
	for _, it := range items {
		rows = append(rows, FlattenValidityRow(it))
	}
	return json.Marshal(map[string]interface{}{"results": rows, "next_cursor": next})
}

// FlattenValidityRow folds a job row's stored report (JSON, usually encoded as a
// string under "report") into the row. The row's own fields win: its status is
// the four-bucket verdict the job counts and filters by, while the report's own
// status (deliverable | undeliverable | unknown) is kept as report_status.
func FlattenValidityRow(row map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(row)+8)
	for k, v := range row {
		out[k] = v
	}
	raw, present := out["report"]
	delete(out, "report")
	report := decodeReport(raw)
	if !present || report == nil {
		return out
	}
	for k, v := range report {
		switch k {
		case "status":
			out["report_status"] = v
		default:
			if _, taken := out[k]; !taken {
				out[k] = v
			}
		}
	}
	return out
}

// decodeReport accepts a report as an object or as a string holding one.
func decodeReport(v interface{}) map[string]interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		return t
	case string:
		var m map[string]interface{}
		if s := strings.TrimSpace(t); s != "" && json.Unmarshal([]byte(s), &m) == nil {
			return m
		}
	}
	return nil
}

// DownloadValidityJob downloads job results as raw CSV or JSON bytes.
// GET /api/cli/jobs/{id}/download?status=&format=
func (c *Client) DownloadValidityJob(ctx context.Context, id, status, format string) ([]byte, error) {
	q := url.Values{}
	if status != "" {
		q.Set("status", status)
	}
	if format != "" {
		q.Set("format", format)
	}
	return c.getBytes(ctx, jobPath(id, "download"), q)
}

// CancelValidityJob cancels a running job.
// POST /api/cli/jobs/{id}/cancel
func (c *Client) CancelValidityJob(ctx context.Context, id string) (json.RawMessage, error) {
	return c.postQuery(ctx, jobPath(id, "cancel"), nil, nil)
}

// ParseJob decodes a job from an API response, whether it is bare, nested under
// a "job" key, or the result of a {success, result, message} envelope.
func ParseJob(data json.RawMessage) (*Job, error) {
	var job Job
	if err := json.Unmarshal(bytes.TrimSpace(unwrapEnvelope(data)), &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// JobIsTerminal reports whether a status represents a finished job.
func JobIsTerminal(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "complete", "done", "finished", "failed", "error", "cancelled", "canceled", "partially_done":
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
