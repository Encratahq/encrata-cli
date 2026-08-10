package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

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

// postWithHeaders issues a POST with extra request headers (e.g. Idempotency-Key)
// on top of the standard auth/content-type headers. No retry: callers that need
// safe retries should supply an Idempotency-Key.
func (c *Client) postWithHeaders(ctx context.Context, path string, payload interface{}, headers map[string]string) (json.RawMessage, error) {
	var reader io.Reader
	hasBody := payload != nil
	if hasBody {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to encode request: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req, hasBody)
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	data, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("failed to read response: %w", readErr)
	}
	if resp.StatusCode >= 400 {
		return nil, parseError(resp.StatusCode, data)
	}
	return json.RawMessage(data), nil
}

// postMultipart sends a multipart/form-data POST with a single file field plus
// optional text fields. Used for large file uploads (e.g. job CSVs).
func (c *Client) postMultipart(ctx context.Context, path, fileField, fileName string, fileData []byte, fields map[string]string) (json.RawMessage, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for key, value := range fields {
		if err := w.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("failed to encode form field: %w", err)
		}
	}
	part, err := w.CreateFormFile(fileField, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to build upload: %w", err)
	}
	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("failed to write upload: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize upload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, parseError(resp.StatusCode, data)
	}
	return json.RawMessage(data), nil
}

// getBytes issues a GET and returns the raw response body, for downloading
// generated files (CSV/JSON) rather than parsing JSON.
func (c *Client) getBytes(ctx context.Context, path string, query url.Values) ([]byte, error) {
	endpoint := c.BaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req, false)

	resp, err := c.streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, parseError(resp.StatusCode, data)
	}
	return data, nil
}
