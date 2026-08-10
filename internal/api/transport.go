package api

import (
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

func (c *Client) patchQuery(ctx context.Context, path string, query url.Values, payload interface{}) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPatch, path, query, payload)
}

func (c *Client) del(ctx context.Context, path string, query url.Values, payload interface{}) (json.RawMessage, error) {
	return c.do(ctx, http.MethodDelete, path, query, payload)
}
