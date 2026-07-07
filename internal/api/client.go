package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Version is stamped into the User-Agent header. Overridden by the cmd package.
var Version = "dev"

const (
	requestTimeout = 90 * time.Second
	streamTimeout  = 5 * time.Minute
	maxRetries     = 3
	initialBackoff = 500 * time.Millisecond
	maxBackoff     = 10 * time.Second
)

var retryableStatus = map[int]bool{
	http.StatusTooManyRequests:     true,
	http.StatusInternalServerError: true,
	http.StatusBadGateway:          true,
	http.StatusServiceUnavailable:  true,
	http.StatusGatewayTimeout:      true,
}

type Client struct {
	BaseURL      string
	APIKey       string
	UserAgent    string
	HTTPClient   *http.Client
	streamClient *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL:      strings.TrimRight(baseURL, "/"),
		APIKey:       apiKey,
		UserAgent:    "encrata-cli/" + Version,
		HTTPClient:   &http.Client{Timeout: requestTimeout},
		streamClient: &http.Client{Timeout: streamTimeout},
	}
}

// Error is an API-level failure carrying the HTTP status code.
type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("request failed with status %d", e.StatusCode)
}

func (c *Client) setHeaders(req *http.Request, hasBody bool) {
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("User-Agent", c.UserAgent)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, payload interface{}) (json.RawMessage, error) {
	var body []byte
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to encode request: %w", err)
		}
		body = b
	}

	endpoint := c.BaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		c.setHeaders(req, body != nil)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("request failed: %w", err)
			if attempt < maxRetries {
				if err := sleepCtx(ctx, retryDelay(attempt, "")); err != nil {
					return nil, err
				}
				continue
			}
			return nil, lastErr
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("failed to read response: %w", readErr)
			if attempt < maxRetries {
				if err := sleepCtx(ctx, retryDelay(attempt, "")); err != nil {
					return nil, err
				}
				continue
			}
			return nil, lastErr
		}

		if retryableStatus[resp.StatusCode] && attempt < maxRetries {
			lastErr = parseError(resp.StatusCode, data)
			if err := sleepCtx(ctx, retryDelay(attempt, resp.Header.Get("Retry-After"))); err != nil {
				return nil, err
			}
			continue
		}

		if resp.StatusCode >= 400 {
			return nil, parseError(resp.StatusCode, data)
		}

		return json.RawMessage(data), nil
	}

	return nil, lastErr
}

// sleepCtx waits for d or until ctx is cancelled, whichever comes first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// stream issues a POST and invokes onEvent for each server-sent data line.
func (c *Client) stream(ctx context.Context, path string, payload interface{}, onEvent func(eventType string, data json.RawMessage) error) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req, true)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.streamClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return parseError(resp.StatusCode, data)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	eventType := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			// Blank line marks the end of an SSE event; reset the type.
			eventType = ""
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(line[len("event:"):])
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		event := strings.TrimSpace(line[len("data:"):])
		if event == "" {
			continue
		}
		if event == "[DONE]" {
			break
		}
		if err := onEvent(eventType, json.RawMessage(event)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (c *Client) post(ctx context.Context, path string, payload interface{}) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, path, nil, payload)
}

func (c *Client) postQuery(ctx context.Context, path string, query url.Values, payload interface{}) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, path, query, payload)
}

func (c *Client) get(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, path, query, nil)
}

func (c *Client) put(ctx context.Context, path string, payload interface{}) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPut, path, nil, payload)
}

func (c *Client) del(ctx context.Context, path string, query url.Values, payload interface{}) (json.RawMessage, error) {
	return c.do(ctx, http.MethodDelete, path, query, payload)
}

func retryDelay(attempt int, retryAfter string) time.Duration {
	if retryAfter != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && secs >= 0 {
			if d := time.Duration(secs) * time.Second; d < maxBackoff {
				return d
			}
			return maxBackoff
		}
	}

	ceiling := initialBackoff * time.Duration(1<<attempt)
	if ceiling > maxBackoff {
		ceiling = maxBackoff
	}
	return time.Duration(rand.Int63n(int64(ceiling) + 1))
}

func parseError(status int, data []byte) error {
	var body struct {
		Error   interface{} `json:"error"`
		Message interface{} `json:"message"`
		Detail  interface{} `json:"detail"`
		Details interface{} `json:"details"`
		Errors  interface{} `json:"errors"`
	}
	msg := ""
	if json.Unmarshal(data, &body) == nil {
		msg = firstErrorText(body.Message, body.Error, body.Detail, body.Details, body.Errors)
	}

	if msg == "" {
		switch status {
		case http.StatusUnauthorized:
			msg = "authentication failed — check your API key"
		case http.StatusPaymentRequired:
			msg = "insufficient credits"
		case http.StatusBadRequest:
			msg = "invalid request"
		case http.StatusTooManyRequests:
			msg = "rate limited — please wait and try again"
		default:
			msg = fmt.Sprintf("request failed with status %d", status)
		}
	}

	return &Error{StatusCode: status, Message: msg}
}

func firstErrorText(values ...interface{}) string {
	for _, value := range values {
		if text := errorText(value); text != "" {
			return text
		}
	}
	return ""
}

func errorText(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []interface{}:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if text := errorText(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "; ")
	case map[string]interface{}:
		for _, key := range []string{"message", "msg", "error", "detail", "field"} {
			if text := errorText(v[key]); text != "" {
				return text
			}
		}
		parts := make([]string, 0, len(v))
		for key, item := range v {
			if text := errorText(item); text != "" {
				parts = append(parts, fmt.Sprintf("%s: %s", key, text))
			}
		}
		return strings.Join(parts, "; ")
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}
