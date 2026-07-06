package api

import (
	"context"
	"encoding/json"
	"net/url"
)

type ListCreateRequest struct {
	Name    string   `json:"name"`
	Type    string   `json:"type,omitempty"`
	Targets []string `json:"targets,omitempty"`
}

func (c *Client) ListContactLists(ctx context.Context, listType string) (json.RawMessage, error) {
	q := url.Values{}
	if listType != "" {
		q.Set("type", listType)
	}
	return c.get(ctx, "/api/agent/lists", q)
}

func (c *Client) CreateContactList(ctx context.Context, req *ListCreateRequest) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/lists", req)
}

func (c *Client) GetContactList(ctx context.Context, id string) (json.RawMessage, error) {
	return c.get(ctx, "/api/agent/lists/"+url.PathEscape(id), nil)
}

func (c *Client) DeleteContactList(ctx context.Context, id string) (json.RawMessage, error) {
	return c.del(ctx, "/api/agent/lists/"+url.PathEscape(id), nil, nil)
}

func (c *Client) ListContactListEmails(ctx context.Context, id string) (json.RawMessage, error) {
	return c.get(ctx, "/api/agent/lists/"+url.PathEscape(id)+"/emails", nil)
}

func (c *Client) AddContactListEmails(ctx context.Context, id string, emails []string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/lists/"+url.PathEscape(id)+"/emails", map[string][]string{"emails": emails})
}

func (c *Client) DeleteContactListEmails(ctx context.Context, id string, emails []string) (json.RawMessage, error) {
	return c.del(ctx, "/api/agent/lists/"+url.PathEscape(id)+"/emails", nil, map[string][]string{"emails": emails})
}
