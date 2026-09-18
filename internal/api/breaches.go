package api

import (
	"context"
	"encoding/json"
	"net/url"
)

// GitHubScanConfig mirrors the owning service's `config` object. Only the knobs
// the CLI exposes are set; anything omitted takes the server's default.
type GitHubScanConfig struct {
	DeepScan        bool     `json:"deep_scan,omitempty"`
	RunAtBackground bool     `json:"run_at_background,omitempty"`
	MinSeverity     string   `json:"min_severity,omitempty"`
	Ref             string   `json:"ref,omitempty"`
	Rules           []string `json:"rules,omitempty"`
}

// GitHubScanRequest is the body of a GitHub repository secret scan.
type GitHubScanRequest struct {
	Repo   string           `json:"repo"`
	Type   *int             `json:"type,omitempty"`
	Config GitHubScanConfig `json:"config"`
}

// ScanGitHubRepo starts a secret scan of a GitHub repository. A deep (full
// history) scan runs in the background and comes back with status "processing".
// POST /api/cli/breaches/github
func (c *Client) ScanGitHubRepo(ctx context.Context, req GitHubScanRequest) (json.RawMessage, error) {
	return c.post(ctx, "/api/cli/breaches/github", req)
}

// GetGitHubScan fetches a previously started GitHub secret scan by its id.
// GET /api/cli/breaches/github/{id}
func (c *Client) GetGitHubScan(ctx context.Context, id string) (json.RawMessage, error) {
	return c.get(ctx, "/api/cli/breaches/github/"+url.PathEscape(id), nil)
}
