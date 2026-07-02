package api

import (
	"bufio"
	"bytes"
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
	BaseURL    string
	APIKey     string
	UserAgent  string
	HTTPClient *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIKey:     apiKey,
		UserAgent:  "encrata-cli/" + Version,
		HTTPClient: &http.Client{Timeout: requestTimeout},
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

func (c *Client) do(method, path string, query url.Values, payload interface{}) (json.RawMessage, error) {
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

		req, err := http.NewRequest(method, endpoint, reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		c.setHeaders(req, body != nil)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if attempt < maxRetries {
				time.Sleep(retryDelay(attempt, ""))
				continue
			}
			return nil, lastErr
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("failed to read response: %w", readErr)
			if attempt < maxRetries {
				time.Sleep(retryDelay(attempt, ""))
				continue
			}
			return nil, lastErr
		}

		if retryableStatus[resp.StatusCode] && attempt < maxRetries {
			lastErr = parseError(resp.StatusCode, data)
			time.Sleep(retryDelay(attempt, resp.Header.Get("Retry-After")))
			continue
		}

		if resp.StatusCode >= 400 {
			return nil, parseError(resp.StatusCode, data)
		}

		return json.RawMessage(data), nil
	}

	return nil, lastErr
}

// stream issues a POST and invokes onEvent for each server-sent data line.
func (c *Client) stream(path string, payload interface{}, onEvent func(json.RawMessage) error) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req, true)
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: streamTimeout}
	resp, err := client.Do(req)
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
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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
		if err := onEvent(json.RawMessage(event)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (c *Client) post(path string, payload interface{}) (json.RawMessage, error) {
	return c.do(http.MethodPost, path, nil, payload)
}

func (c *Client) postQuery(path string, query url.Values, payload interface{}) (json.RawMessage, error) {
	return c.do(http.MethodPost, path, query, payload)
}

func (c *Client) get(path string, query url.Values) (json.RawMessage, error) {
	return c.do(http.MethodGet, path, query, nil)
}

func (c *Client) put(path string, payload interface{}) (json.RawMessage, error) {
	return c.do(http.MethodPut, path, nil, payload)
}

func (c *Client) del(path string, query url.Values, payload interface{}) (json.RawMessage, error) {
	return c.do(http.MethodDelete, path, query, payload)
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
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	msg := ""
	if json.Unmarshal(data, &body) == nil {
		if body.Message != "" {
			msg = body.Message
		} else if body.Error != "" {
			msg = body.Error
		}
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
