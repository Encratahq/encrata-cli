package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Response decoding for breaches jobs lives here (not in cmd) so the move to the
// unified /api/jobs surface and its {success,result,message} envelope is a change
// to this package only.

// BreachesJob is the status view of an async bulk email-breaches job.
type BreachesJob struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	FileName       string `json:"file_name,omitempty"`
	TotalEmails    int    `json:"total_emails"`
	ProcessedCount int    `json:"processed_count"`
	BreachedCount  int    `json:"breached_count"`
	CreditsUsed    int    `json:"credits_used"`
	ErrorMessage   string `json:"error_message,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
}

// BreachesJobItem is one resolved address from a breaches job. Report is the
// full per-email breach report (breach_count, services, ...), the same object
// the bulk-breaches stream returns under breach_info.
type BreachesJobItem struct {
	Email       string
	Breached    bool
	BreachCount int
	Services    string
	Report      json.RawMessage
}

const (
	breachesPollMin  = 2 * time.Second
	breachesPollMax  = 5 * time.Second
	breachesPageSize = 200 // the server caps results pages at 200 rows
)

// unwrapEnvelope returns the payload of a {"job": {...}} or
// {"success": ..., "result": {...}} response, or data unchanged when bare.
func unwrapEnvelope(data json.RawMessage) json.RawMessage {
	var wrap struct {
		Success *bool           `json:"success"`
		Result  json.RawMessage `json:"result"`
		Job     json.RawMessage `json:"job"`
	}
	if json.Unmarshal(data, &wrap) != nil {
		return data
	}
	if isJSONObject(wrap.Job) {
		return wrap.Job
	}
	if wrap.Success != nil && isJSONObject(wrap.Result) {
		return wrap.Result
	}
	return data
}

func isJSONObject(raw []byte) bool {
	t := bytes.TrimSpace(raw)
	return len(t) > 0 && t[0] == '{'
}

// ParseBreachesJob decodes a breaches job from a create/status response.
func ParseBreachesJob(data json.RawMessage) (*BreachesJob, error) {
	var job BreachesJob
	if err := json.Unmarshal(unwrapEnvelope(data), &job); err != nil {
		return nil, fmt.Errorf("failed to decode breaches job: %w", err)
	}
	return &job, nil
}

// nextPollInterval backs a poll interval off by 1.5x, capped at breachesPollMax.
func nextPollInterval(d time.Duration) time.Duration {
	d = d * 3 / 2
	if d > breachesPollMax {
		return breachesPollMax
	}
	return d
}

// PollBreachesJob polls a breaches job until it reaches a terminal state,
// calling onUpdate after each poll. The interval starts at 2s and backs off to
// 5s. Cancelling ctx stops polling only; the job keeps running server-side.
func (c *Client) PollBreachesJob(ctx context.Context, id string, onUpdate func(*BreachesJob)) (*BreachesJob, error) {
	interval := breachesPollMin
	for {
		data, err := c.GetBreachesJob(ctx, id)
		if err != nil {
			return nil, err
		}
		job, err := ParseBreachesJob(data)
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
		interval = nextPollInterval(interval)
	}
}

// breachesItemWire is a results row as served by the jobs API: data holds the
// report JSON-encoded as a string and services is a display summary.
type breachesItemWire struct {
	Email       string          `json:"email"`
	Breached    bool            `json:"breached"`
	BreachCount int             `json:"breach_count"`
	Services    json.RawMessage `json:"services"`
	Data        json.RawMessage `json:"data"`
}

func (w breachesItemWire) item() BreachesJobItem {
	it := BreachesJobItem{Email: w.Email, Breached: w.Breached, BreachCount: w.BreachCount}
	var s string
	if json.Unmarshal(w.Services, &s) == nil {
		it.Services = s
	}
	it.Report = decodeEmbeddedJSON(w.Data)
	return it
}

// decodeEmbeddedJSON accepts a JSON object, or a string holding one.
func decodeEmbeddedJSON(raw json.RawMessage) json.RawMessage {
	if isJSONObject(raw) {
		return raw
	}
	var s string
	if json.Unmarshal(raw, &s) == nil && isJSONObject([]byte(s)) && json.Valid([]byte(s)) {
		return json.RawMessage(s)
	}
	return nil
}

type breachesItemsPage struct {
	Items      []breachesItemWire `json:"items"`
	NextCursor *string            `json:"next_cursor"`
}

func decodeBreachesItemsPage(data json.RawMessage) (*breachesItemsPage, error) {
	var p breachesItemsPage
	if err := json.Unmarshal(unwrapEnvelope(data), &p); err != nil {
		return nil, fmt.Errorf("failed to decode breaches results: %w", err)
	}
	return &p, nil
}

func (p *breachesItemsPage) items() []BreachesJobItem {
	out := make([]BreachesJobItem, 0, len(p.Items))
	for _, w := range p.Items {
		out = append(out, w.item())
	}
	return out
}

// FetchBreachesJobItems walks every result row of a completed breaches job,
// calling onPage once per page, using keyset (cursor) paging — the only mode the
// unified results route serves.
// GET /api/cli/jobs/{id}?type=breaches&cursor=&page_size=&breached=1
func (c *Client) FetchBreachesJobItems(ctx context.Context, id string, breachedOnly bool, onPage func([]BreachesJobItem) error) error {
	cursor := "0" // the first keyset page
	for {
		data, err := c.breachesResultsPage(ctx, id, cursor, breachesPageSize, breachedOnly)
		if err != nil {
			return err
		}
		p, err := decodeBreachesItemsPage(data)
		if err != nil {
			return err
		}
		if err := onPage(p.items()); err != nil {
			return err
		}
		if p.NextCursor == nil || *p.NextCursor == "" || *p.NextCursor == cursor || len(p.Items) == 0 {
			return nil
		}
		cursor = *p.NextCursor
	}
}
