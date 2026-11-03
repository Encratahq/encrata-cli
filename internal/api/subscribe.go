package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

// Terminal bulk-job events sent by GET /api/cli/subscribe. Progress is never sent.
const (
	EventJobCompleted = "job.completed"
	EventJobFailed    = "job.failed"
	EventJobCancelled = "job.cancelled"
)

const (
	subscribePath = "/api/cli/subscribe"

	// subscribeIdle is how long a connection may stay silent before it is treated
	// as dead and redialled. The server pings every 20s, so this is three misses.
	subscribeIdle = 60 * time.Second

	subscribeBackoffMin = time.Second
	subscribeBackoffMax = 30 * time.Second

	// subscribeDedupeCap bounds the at-least-once dedupe set for long follows.
	subscribeDedupeCap = 4096
)

// ErrStopSubscribe, returned from a Subscribe callback, ends the subscription
// cleanly: Subscribe returns nil.
var ErrStopSubscribe = errors.New("stop subscription")

// JobEvent is one terminal bulk-job frame from the completion stream. The stream
// is user-scoped, so an event may belong to a job run in another workspace.
type JobEvent struct {
	Event     string         `json:"event"`
	JobID     string         `json:"job_id"`
	Kind      string         `json:"kind"`
	Status    string         `json:"status"`
	Processed int            `json:"processed"`
	Total     int            `json:"total"`
	Counts    map[string]int `json:"counts,omitempty"`
}

// Succeeded reports whether the job completed (rather than failed or was cancelled).
func (e JobEvent) Succeeded() bool { return e.Event == EventJobCompleted }

// SubscribeHooks are optional connection-state callbacks for Subscribe.
type SubscribeHooks struct {
	// OnConnect fires each time the stream is (re)established.
	OnConnect func()
	// OnRetry fires after a dropped or failed connection, before waiting to redial.
	OnRetry func(err error, wait time.Duration)
}

// Subscribe holds GET /api/cli/subscribe open and calls onEvent for every
// terminal bulk-job frame until ctx is cancelled or onEvent returns an error
// (ErrStopSubscribe ends it cleanly). A dropped connection is redialled with
// jittered exponential backoff; frames are treated as at-least-once and
// deduplicated by job_id + event. Auth and not-found responses are not retried.
func (c *Client) Subscribe(ctx context.Context, hooks SubscribeHooks, onEvent func(JobEvent) error) error {
	seen := make(map[string]struct{})
	deliver := func(ev JobEvent) error {
		key := ev.Event + "|" + ev.JobID
		if _, dup := seen[key]; dup {
			return nil
		}
		if len(seen) >= subscribeDedupeCap {
			clear(seen)
		}
		seen[key] = struct{}{}
		return onEvent(ev)
	}

	backoff := subscribeBackoffMin
	for {
		connected, err := c.subscribeOnce(ctx, hooks.OnConnect, deliver)
		if errors.Is(err, ErrStopSubscribe) {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var apiErr *Error
		if errors.As(err, &apiErr) && !retryableStatus[apiErr.StatusCode] {
			return err
		}
		var cbErr *callbackError
		if errors.As(err, &cbErr) {
			return cbErr.err
		}
		if connected {
			backoff = subscribeBackoffMin
		}
		wait := backoff + rand.N(backoff/5+1)
		if hooks.OnRetry != nil {
			hooks.OnRetry(err, wait)
		}
		if err := sleepCtx(ctx, wait); err != nil {
			return err
		}
		backoff = min(backoff*2, subscribeBackoffMax)
	}
}

// callbackError marks an error returned by the caller's onEvent, so Subscribe
// hands it back instead of reconnecting.
type callbackError struct{ err error }

func (e *callbackError) Error() string { return e.err.Error() }
func (e *callbackError) Unwrap() error { return e.err }

// subscribeOnce runs one connection. connected reports whether the stream was
// established, so the caller can reset its backoff.
func (c *Client) subscribeOnce(ctx context.Context, onConnect func(), onEvent func(JobEvent) error) (connected bool, err error) {
	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	req, err := http.NewRequestWithContext(connCtx, http.MethodGet, c.BaseURL+subscribePath, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req, false)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.subscribeClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return false, parseError(resp.StatusCode, data)
	}
	if onConnect != nil {
		onConnect()
	}

	// Watchdog: any line (frame or ": ping") proves the connection is alive.
	idle := time.AfterFunc(subscribeIdle, cancel)
	defer idle.Stop()

	err = readSSE(resp.Body, func() { idle.Reset(subscribeIdle) }, func(event, data string) error {
		ev, ok := decodeJobEvent(event, data)
		if !ok {
			return nil
		}
		if err := onEvent(ev); err != nil {
			if errors.Is(err, ErrStopSubscribe) {
				return err
			}
			return &callbackError{err}
		}
		return nil
	})
	if err == nil {
		err = io.ErrUnexpectedEOF // the server never ends the stream on its own
	}
	if ctx.Err() == nil && connCtx.Err() != nil {
		err = fmt.Errorf("no data for %s", subscribeIdle)
	}
	return true, err
}

// decodeJobEvent turns one SSE frame into a JobEvent, skipping anything that is
// not a terminal job event.
func decodeJobEvent(event, data string) (JobEvent, bool) {
	switch event {
	case EventJobCompleted, EventJobFailed, EventJobCancelled:
	default:
		return JobEvent{}, false
	}
	var ev JobEvent
	if json.Unmarshal([]byte(data), &ev) != nil || ev.JobID == "" {
		return JobEvent{}, false
	}
	ev.Event = event
	return ev, true
}

// readSSE parses a text/event-stream body, calling onLine for every line read
// and onFrame once per dispatched event (multi-line data joined by "\n").
// Comment lines (": ping") only count as activity.
func readSSE(r io.Reader, onLine func(), onFrame func(event, data string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var event string
	var data []string
	for scanner.Scan() {
		onLine()
		line := strings.TrimRight(scanner.Text(), "\r")
		switch {
		case line == "":
			if len(data) > 0 {
				if err := onFrame(event, strings.Join(data, "\n")); err != nil {
					return err
				}
			}
			event, data = "", nil
		case strings.HasPrefix(line, ":"):
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(line[len("event:"):])
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(line[len("data:"):], " "))
		}
	}
	return scanner.Err()
}
