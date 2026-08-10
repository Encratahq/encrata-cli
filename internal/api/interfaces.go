package api

import (
	"context"
	"encoding/json"
	"time"
)

// This file defines segregated, consumer-facing interfaces implemented by
// *Client. Commands depend on these abstractions (DIP) and each interface stays
// small and focused (ISP), which also makes commands unit-testable with fakes.

// AccountAPI exposes account/identity endpoints.
type AccountAPI interface {
	Me(ctx context.Context) (json.RawMessage, error)
}

// EmailAPI exposes single- and bulk-email intelligence lookups.
type EmailAPI interface {
	EmailValidity(ctx context.Context, email string) (json.RawMessage, error)
	EmailEnrich(ctx context.Context, email string) (json.RawMessage, error)
	EmailIdentity(ctx context.Context, email string) (json.RawMessage, error)
	EmailBreaches(ctx context.Context, email string) (json.RawMessage, error)
	EmailVerify(ctx context.Context, email string) (json.RawMessage, error)
	EmailValidityBulk(ctx context.Context, emails []string) (json.RawMessage, error)
}

// PasswordAPI exposes password breach checks (SHA-1 only).
type PasswordAPI interface {
	PasswordBreaches(ctx context.Context, sha1 string) (json.RawMessage, error)
	PasswordBreachesBulk(ctx context.Context, sha1s []string) (json.RawMessage, error)
}

// ValidityJobAPI covers async validity jobs.
type ValidityJobAPI interface {
	CreateValidityJob(ctx context.Context, req *CreateJobRequest) (json.RawMessage, error)
	CreateValidityJobFile(ctx context.Context, fileName string, data []byte) (json.RawMessage, error)
	GetValidityJob(ctx context.Context, id string) (json.RawMessage, error)
	ListValidityJobs(ctx context.Context) (json.RawMessage, error)
	GetValidityJobResults(ctx context.Context, id string, page int, status string) (json.RawMessage, error)
	GetValidityJobResultsPage(ctx context.Context, id string, page, pageSize int) (json.RawMessage, error)
	DownloadValidityJob(ctx context.Context, id, status, format string) ([]byte, error)
	CancelValidityJob(ctx context.Context, id string) (json.RawMessage, error)
	RetryValidityJob(ctx context.Context, id string) (json.RawMessage, error)
	PollValidityJob(ctx context.Context, id string, interval time.Duration, onUpdate func(*Job)) (*Job, error)
}

// IdentityJobAPI covers async identity jobs.
type IdentityJobAPI interface {
	CreateIdentityJob(ctx context.Context, emails []string, fileName string) (json.RawMessage, error)
	GetIdentityJob(ctx context.Context, id string) (json.RawMessage, error)
	ListIdentityJobs(ctx context.Context) (json.RawMessage, error)
	GetIdentityJobResults(ctx context.Context, id string, page, pageSize int, foundOnly bool) (json.RawMessage, error)
	DownloadIdentityJob(ctx context.Context, id string, foundOnly bool) ([]byte, error)
	CancelIdentityJob(ctx context.Context, id string) (json.RawMessage, error)
	RetryIdentityJob(ctx context.Context, id string) (json.RawMessage, error)
}

// PasswordJobAPI covers async password-breach jobs.
type PasswordJobAPI interface {
	CreatePasswordJob(ctx context.Context, sha1s []string, fileName string) (json.RawMessage, error)
	GetPasswordJob(ctx context.Context, id string) (json.RawMessage, error)
	ListPasswordJobs(ctx context.Context) (json.RawMessage, error)
	GetPasswordJobResults(ctx context.Context, id string, page, pageSize int, breachedOnly bool) (json.RawMessage, error)
	DownloadPasswordJob(ctx context.Context, id string, breachedOnly bool) ([]byte, error)
	CancelPasswordJob(ctx context.Context, id string) (json.RawMessage, error)
	RetryPasswordJob(ctx context.Context, id string) (json.RawMessage, error)
}

// BulkJobAPI covers generic bulk-job management.
type BulkJobAPI interface {
	ListBulkJobs(ctx context.Context) (json.RawMessage, error)
	GetBulkJob(ctx context.Context, id string) (json.RawMessage, error)
	CancelBulkJob(ctx context.Context, id string) (json.RawMessage, error)
}

// JobsAPI aggregates every async-job surface.
type JobsAPI interface {
	ValidityJobAPI
	IdentityJobAPI
	PasswordJobAPI
	BulkJobAPI
}

// KeysAPI exposes API-key management.
type KeysAPI interface {
	ListKeys(ctx context.Context) (json.RawMessage, error)
	CreateKey(ctx context.Context, name string) (json.RawMessage, error)
	RevokeKey(ctx context.Context, id string, permanent bool) (json.RawMessage, error)
	RenameKey(ctx context.Context, id, name string) (json.RawMessage, error)
	SetKeyStatus(ctx context.Context, id, action string) (json.RawMessage, error)
	SetKeyCreditLimit(ctx context.Context, id string, limit *int) (json.RawMessage, error)
}

// ListsAPI exposes contact-list management.
type ListsAPI interface {
	ListContactLists(ctx context.Context) (json.RawMessage, error)
	CreateContactList(ctx context.Context, name string, emails []string) (json.RawMessage, error)
	GetContactList(ctx context.Context, id string) (json.RawMessage, error)
	DeleteContactList(ctx context.Context, id string) (json.RawMessage, error)
	ListContactListEmails(ctx context.Context, id string) (json.RawMessage, error)
	AddEmailsToList(ctx context.Context, id string, emails []string) (json.RawMessage, error)
	RemoveEmailsFromList(ctx context.Context, id string, emails []string) (json.RawMessage, error)
}

// WebhooksAPI exposes webhook management.
type WebhooksAPI interface {
	ListWebhooks(ctx context.Context) (json.RawMessage, error)
	GetWebhook(ctx context.Context, id string) (json.RawMessage, error)
	CreateWebhook(ctx context.Context, targetURL, description string, events []string) (json.RawMessage, error)
	UpdateWebhook(ctx context.Context, id, targetURL, description string, events []string, isActive *bool) (json.RawMessage, error)
	DeleteWebhook(ctx context.Context, id string) (json.RawMessage, error)
	TestWebhook(ctx context.Context, id string) (json.RawMessage, error)
	ListWebhookDeliveries(ctx context.Context, webhookID string) (json.RawMessage, error)
}

// WorkflowsAPI exposes workflow run management.
type WorkflowsAPI interface {
	UploadWorkflowFile(ctx context.Context, fileName string, data []byte, workflowID string) (json.RawMessage, error)
	RunWorkflow(ctx context.Context, workflowID, fileID, idempotencyKey string) (json.RawMessage, error)
	GetWorkflowRun(ctx context.Context, runID string) (json.RawMessage, error)
	ListWorkflowRuns(ctx context.Context, workflowID string, limit, offset int) (json.RawMessage, error)
	CancelWorkflowRun(ctx context.Context, runID string) (json.RawMessage, error)
	DownloadWorkflowRunOutput(ctx context.Context, runID string) ([]byte, error)
}

// IntegrationsAPI exposes connected export destinations.
type IntegrationsAPI interface {
	ListIntegrations(ctx context.Context) (json.RawMessage, error)
	NangoProviders(ctx context.Context) (json.RawMessage, error)
	NangoSession(ctx context.Context, integration string) (json.RawMessage, error)
	NangoSave(ctx context.Context, connectionID, providerConfigKey, label string) (json.RawMessage, error)
	CreateIntegrationSheet(ctx context.Context, id, title string) (json.RawMessage, error)
	DeleteIntegration(ctx context.Context, id string) (json.RawMessage, error)
}

// WorkspaceAPI exposes workspace and member management.
type WorkspaceAPI interface {
	ListWorkspaces(ctx context.Context) (json.RawMessage, error)
	CreateWorkspace(ctx context.Context, name, slug string, logoURL *string) (json.RawMessage, error)
	SwitchWorkspace(ctx context.Context, id string) (json.RawMessage, error)
	UpdateWorkspace(ctx context.Context, id, name, slug string, logoURL *string) (json.RawMessage, error)
	DeleteWorkspace(ctx context.Context) (json.RawMessage, error)
	ListWorkspaceMembers(ctx context.Context) (json.RawMessage, error)
	InviteWorkspaceMember(ctx context.Context, email, role string) (json.RawMessage, error)
	SetWorkspaceMemberRole(ctx context.Context, memberID, role string) (json.RawMessage, error)
	RemoveWorkspaceMember(ctx context.Context, memberID string) (json.RawMessage, error)
}

// BulkSearchAPI exposes streaming bulk search.
type BulkSearchAPI interface {
	BulkValiditySearch(ctx context.Context, queries []string, fileName string, onEvent func(BulkEvent) error) error
	BulkBreachesSearch(ctx context.Context, queries []string, fileName string, onEvent func(BulkEvent) error) error
}

// API aggregates every segregated interface; *Client is the sole implementation.
type API interface {
	AccountAPI
	EmailAPI
	PasswordAPI
	JobsAPI
	KeysAPI
	ListsAPI
	WebhooksAPI
	WorkflowsAPI
	IntegrationsAPI
	WorkspaceAPI
	BulkSearchAPI
}

var _ API = (*Client)(nil)
