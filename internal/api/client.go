package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) post(endpoint string, payload interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited — please wait and try again")
	}

	if resp.StatusCode != 200 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	return data, nil
}

// ── Email Lookup ──────────────────────────────────────────

type EmailRequest struct {
	Email   string   `json:"email"`
	Country string   `json:"country,omitempty"`
	Lang    string   `json:"lang,omitempty"`
	Num     int      `json:"num,omitempty"`
	Page    int      `json:"page,omitempty"`
	Fields  []string `json:"fields,omitempty"`
}

func (c *Client) EmailLookup(req *EmailRequest) (json.RawMessage, error) {
	data, err := c.post("/api/agent/lookup", req)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// ── Google Search ─────────────────────────────────────────

type GoogleRequest struct {
	Query   string `json:"query"`
	Type    string `json:"type,omitempty"`
	Country string `json:"country,omitempty"`
	Lang    string `json:"lang,omitempty"`
	Num     int    `json:"num,omitempty"`
	Page    int    `json:"page,omitempty"`
}

func (c *Client) GoogleSearch(req *GoogleRequest) (json.RawMessage, error) {
	data, err := c.post("/api/agent/google", req)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// ── Phone Search ──────────────────────────────────────────

type PhoneRequest struct {
	Query string `json:"query"`
}

func (c *Client) PhoneSearch(query string) (json.RawMessage, error) {
	data, err := c.post("/api/agent/phone", &PhoneRequest{Query: query})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// ── Company Search ────────────────────────────────────────

type CompanyRequest struct {
	Query string `json:"query"`
}

func (c *Client) CompanySearch(query string) (json.RawMessage, error) {
	data, err := c.post("/api/agent/company", &CompanyRequest{Query: query})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// ── Domain Search ─────────────────────────────────────────

type DomainRequest struct {
	Query string `json:"query"`
}

func (c *Client) DomainSearch(query string) (json.RawMessage, error) {
	data, err := c.post("/api/agent/domain", &DomainRequest{Query: query})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// ── IP Search ─────────────────────────────────────────────

type IPRequest struct {
	Query string `json:"query"`
}

func (c *Client) IPSearch(query string) (json.RawMessage, error) {
	data, err := c.post("/api/agent/ip", &IPRequest{Query: query})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// ── Darkweb Search ────────────────────────────────────────

type DarkwebRequest struct {
	Query  string `json:"query"`
	Offset int    `json:"offset,omitempty"`
}

func (c *Client) DarkwebSearch(req *DarkwebRequest) (json.RawMessage, error) {
	data, err := c.post("/api/agent/darkweb", req)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}
