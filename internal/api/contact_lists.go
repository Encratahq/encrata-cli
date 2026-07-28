package api

import (
	"context"
	"encoding/json"
	"net/url"
)

// ContactList is a reusable named group of emails.
type ContactList struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	EmailCount int    `json:"email_count"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// ContactListEmail is one email row in a contact list.
type ContactListEmail struct {
	Email string `json:"email"`
}

// ListContactLists lists all contact lists.
// GET /api/agent/lists
func (c *Client) ListContactLists(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/agent/lists", nil)
}

// CreateContactList creates a new contact list and optionally seeds emails.
// POST /api/agent/lists
func (c *Client) CreateContactList(ctx context.Context, name string, emails []string) (json.RawMessage, error) {
	payload := map[string]interface{}{"name": name}
	if len(emails) > 0 {
		payload["emails"] = emails
	}
	return c.post(ctx, "/api/agent/lists", payload)
}

// GetContactList fetches one contact list by id.
// GET /api/agent/lists/:id
func (c *Client) GetContactList(ctx context.Context, id string) (json.RawMessage, error) {
	return c.get(ctx, "/api/agent/lists/"+url.PathEscape(id), nil)
}

// DeleteContactList permanently deletes one contact list by id.
// DELETE /api/agent/lists/:id
func (c *Client) DeleteContactList(ctx context.Context, id string) (json.RawMessage, error) {
	return c.del(ctx, "/api/agent/lists/"+url.PathEscape(id), nil, nil)
}

// ListContactListEmails returns all emails in a list.
// GET /api/agent/lists/:id/emails
func (c *Client) ListContactListEmails(ctx context.Context, id string) (json.RawMessage, error) {
	return c.get(ctx, "/api/agent/lists/"+url.PathEscape(id)+"/emails", nil)
}

// AddEmailsToList adds emails to an existing contact list.
// POST /api/agent/lists/:id/emails
func (c *Client) AddEmailsToList(ctx context.Context, id string, emails []string) (json.RawMessage, error) {
	return c.post(ctx, "/api/agent/lists/"+url.PathEscape(id)+"/emails", map[string][]string{"emails": emails})
}

// RemoveEmailsFromList removes emails from an existing contact list.
// DELETE /api/agent/lists/:id/emails
func (c *Client) RemoveEmailsFromList(ctx context.Context, id string, emails []string) (json.RawMessage, error) {
	return c.del(ctx, "/api/agent/lists/"+url.PathEscape(id)+"/emails", nil, map[string][]string{"emails": emails})
}
