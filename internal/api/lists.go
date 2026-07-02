package api

import (
	"encoding/json"
	"net/url"
)

type ListCreateRequest struct {
	Name    string   `json:"name"`
	Type    string   `json:"type,omitempty"`
	Targets []string `json:"targets,omitempty"`
}

func (c *Client) ListContactLists(listType string) (json.RawMessage, error) {
	q := url.Values{}
	if listType != "" {
		q.Set("type", listType)
	}
	return c.get("/api/agent/lists", q)
}

func (c *Client) CreateContactList(req *ListCreateRequest) (json.RawMessage, error) {
	return c.post("/api/agent/lists", req)
}

func (c *Client) GetContactList(id string) (json.RawMessage, error) {
	return c.get("/api/agent/lists/"+url.PathEscape(id), nil)
}

func (c *Client) DeleteContactList(id string) (json.RawMessage, error) {
	return c.del("/api/agent/lists/"+url.PathEscape(id), nil, nil)
}

func (c *Client) ListContactListEmails(id string) (json.RawMessage, error) {
	return c.get("/api/agent/lists/"+url.PathEscape(id)+"/emails", nil)
}

func (c *Client) AddContactListEmails(id string, emails []string) (json.RawMessage, error) {
	return c.post("/api/agent/lists/"+url.PathEscape(id)+"/emails", map[string][]string{"emails": emails})
}

func (c *Client) DeleteContactListEmails(id string, emails []string) (json.RawMessage, error) {
	return c.del("/api/agent/lists/"+url.PathEscape(id)+"/emails", nil, map[string][]string{"emails": emails})
}
