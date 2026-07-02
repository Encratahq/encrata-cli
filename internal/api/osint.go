package api

import "encoding/json"

type GoogleRequest struct {
	Query   string `json:"query"`
	Type    string `json:"type,omitempty"`
	Country string `json:"country,omitempty"`
	Lang    string `json:"lang,omitempty"`
	Num     int    `json:"num,omitempty"`
	Page    int    `json:"page,omitempty"`
}

func (c *Client) GoogleSearch(req *GoogleRequest) (json.RawMessage, error) {
	return c.post("/api/agent/google", req)
}

type PhoneRequest struct {
	Query string `json:"query"`
}

func (c *Client) PhoneSearch(query string) (json.RawMessage, error) {
	return c.post("/api/agent/phone", &PhoneRequest{Query: query})
}

type CompanyRequest struct {
	Query string `json:"query"`
}

func (c *Client) CompanySearch(query string) (json.RawMessage, error) {
	return c.post("/api/agent/company", &CompanyRequest{Query: query})
}

type DomainRequest struct {
	Query string `json:"query"`
}

func (c *Client) DomainSearch(query string) (json.RawMessage, error) {
	return c.post("/api/agent/domain", &DomainRequest{Query: query})
}

type IPRequest struct {
	Query string `json:"query"`
	IP    string `json:"ip"`
}

func (c *Client) IPSearch(query string) (json.RawMessage, error) {
	return c.post("/api/agent/ip", &IPRequest{Query: query, IP: query})
}

type DarkwebRequest struct {
	Query  string `json:"query"`
	Offset int    `json:"offset,omitempty"`
}

func (c *Client) DarkwebSearch(req *DarkwebRequest) (json.RawMessage, error) {
	return c.post("/api/agent/darkweb", req)
}
