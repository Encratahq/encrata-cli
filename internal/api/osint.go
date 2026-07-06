package api

import (
	"context"
	"encoding/json"
)

type GoogleRequest struct {
	Query   string `json:"query"`
	Type    string `json:"type,omitempty"`
	Country string `json:"country,omitempty"`
	Lang    string `json:"lang,omitempty"`
	Num     int    `json:"num,omitempty"`
	Page    int    `json:"page,omitempty"`
}

func (c *Client) GoogleSearch(ctx context.Context, req *GoogleRequest) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/google", req)
}

type PhoneRequest struct {
	Query string `json:"query"`
}

func (c *Client) PhoneSearch(ctx context.Context, query string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/phone", &PhoneRequest{Query: query})
}

type CompanyRequest struct {
	Query string `json:"query"`
}

func (c *Client) CompanySearch(ctx context.Context, query string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/company", &CompanyRequest{Query: query})
}

type DomainRequest struct {
	Query string `json:"query"`
}

func (c *Client) DomainSearch(ctx context.Context, query string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/domain", &DomainRequest{Query: query})
}

type IPRequest struct {
	Query string `json:"query"`
	IP    string `json:"ip"`
}

func (c *Client) IPSearch(ctx context.Context, query string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/ip", &IPRequest{Query: query, IP: query})
}

type DarkwebRequest struct {
	Query  string `json:"query"`
	Offset int    `json:"offset,omitempty"`
}

func (c *Client) DarkwebSearch(ctx context.Context, req *DarkwebRequest) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/darkweb", req)
}
