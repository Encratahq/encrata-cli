package api

import (
	"context"
	"encoding/json"
	"net/url"
)

// ListWorkspaces returns the workspaces the caller belongs to.
// GET /api/workspaces
func (c *Client) ListWorkspaces(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/workspaces", nil)
}

// CreateWorkspace creates a new workspace. Slug is auto-generated when empty.
// POST /api/workspaces
func (c *Client) CreateWorkspace(ctx context.Context, name, slug string, logoURL *string) (json.RawMessage, error) {
	body := map[string]interface{}{"name": name}
	if slug != "" {
		body["slug"] = slug
	}
	if logoURL != nil {
		body["logo_url"] = *logoURL
	}
	return c.post(ctx, "/api/workspaces", body)
}

// SwitchWorkspace sets the caller's active workspace.
// PUT /api/workspaces/switch
func (c *Client) SwitchWorkspace(ctx context.Context, id string) (json.RawMessage, error) {
	return c.put(ctx, "/api/workspaces/switch", map[string]string{"workspace_id": id})
}

// UpdateWorkspace updates a workspace's name/slug/logo. An empty id targets the
// caller's current workspace. The server regenerates the slug from name when
// slug is empty.
// PUT /api/workspaces
func (c *Client) UpdateWorkspace(ctx context.Context, id, name, slug string, logoURL *string) (json.RawMessage, error) {
	body := map[string]interface{}{"name": name}
	if id != "" {
		body["workspace_id"] = id
	}
	if slug != "" {
		body["slug"] = slug
	}
	if logoURL != nil {
		body["logo_url"] = *logoURL
	}
	return c.put(ctx, "/api/workspaces", body)
}

// DeleteWorkspace deletes the caller's current workspace.
// DELETE /api/workspaces
func (c *Client) DeleteWorkspace(ctx context.Context) (json.RawMessage, error) {
	return c.del(ctx, "/api/workspaces", nil, nil)
}

// ListWorkspaceMembers returns members of the caller's current workspace.
// GET /api/workspaces/members
func (c *Client) ListWorkspaceMembers(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/workspaces/members", nil)
}

// InviteWorkspaceMember invites an email to the current workspace with a role.
// POST /api/workspaces/members
func (c *Client) InviteWorkspaceMember(ctx context.Context, email, role string) (json.RawMessage, error) {
	return c.post(ctx, "/api/workspaces/members", map[string]string{"email": email, "role": role})
}

// SetWorkspaceMemberRole changes a member's role in the current workspace.
// PUT /api/workspaces/members
func (c *Client) SetWorkspaceMemberRole(ctx context.Context, memberID, role string) (json.RawMessage, error) {
	return c.put(ctx, "/api/workspaces/members", map[string]string{"member_id": memberID, "role": role})
}

// RemoveWorkspaceMember removes a member from the current workspace.
// DELETE /api/workspaces/members?id=
func (c *Client) RemoveWorkspaceMember(ctx context.Context, memberID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("id", memberID)
	return c.del(ctx, "/api/workspaces/members", q, nil)
}
