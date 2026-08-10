package cmd

import (
	"context"
	"encoding/json"

	"github.com/Encratahq/cli/internal/api"
	"github.com/spf13/cobra"
)

// jobStrategy encapsulates everything that varies between non-validity async
// job types (identity, password). Adding a new type is one registry entry — no
// command handler needs editing (Open/Closed).
type jobStrategy struct {
	title string

	create   func(cmd *cobra.Command, args []string, c api.JobsAPI) (json.RawMessage, error)
	list     func(ctx context.Context, c api.JobsAPI) (json.RawMessage, error)
	status   func(ctx context.Context, c api.JobsAPI, id string) (json.RawMessage, error)
	results  func(ctx context.Context, c api.JobsAPI, id string, page, pageSize int, breached, foundOnly bool) (json.RawMessage, error)
	download func(ctx context.Context, c api.JobsAPI, id string, breached, foundOnly bool) ([]byte, error)
	cancel   func(ctx context.Context, c api.JobsAPI, id string) (json.RawMessage, error)
	retry    func(ctx context.Context, c api.JobsAPI, id string) (json.RawMessage, error) // nil ⇒ unsupported

	printJob func(map[string]interface{})

	listColumns []string
	listRow     func(map[string]interface{}) []string

	resultsKey    string
	resultColumns []string
	resultRow     func(map[string]interface{}) []string
}

// strategyFor returns the strategy for a validated non-validity job type.
func strategyFor(jt string) jobStrategy {
	return jobStrategies[jt]
}

var jobStrategies = map[string]jobStrategy{
	"identity": {
		title:  "Identity",
		create: createIdentityJob,
		list:   func(ctx context.Context, c api.JobsAPI) (json.RawMessage, error) { return c.ListIdentityJobs(ctx) },
		status: func(ctx context.Context, c api.JobsAPI, id string) (json.RawMessage, error) { return c.GetIdentityJob(ctx, id) },
		results: func(ctx context.Context, c api.JobsAPI, id string, page, pageSize int, breached, foundOnly bool) (json.RawMessage, error) {
			return c.GetIdentityJobResults(ctx, id, page, pageSize, foundOnly || breached)
		},
		download: func(ctx context.Context, c api.JobsAPI, id string, breached, foundOnly bool) ([]byte, error) {
			return c.DownloadIdentityJob(ctx, id, foundOnly || breached)
		},
		cancel:        func(ctx context.Context, c api.JobsAPI, id string) (json.RawMessage, error) { return c.CancelIdentityJob(ctx, id) },
		retry:         func(ctx context.Context, c api.JobsAPI, id string) (json.RawMessage, error) { return c.RetryIdentityJob(ctx, id) },
		printJob:      printIdentityJob,
		listColumns:   []string{"ID", "Status", "Total", "Processed", "Found", "Credits"},
		listRow:       identityListRow,
		resultsKey:    "items",
		resultColumns: []string{"Email", "Found", "Name", "Company", "Role", "Location"},
		resultRow:     identityResultRow,
	},
	"password": {
		title:  "Password",
		create: createPasswordJob,
		list:   func(ctx context.Context, c api.JobsAPI) (json.RawMessage, error) { return c.ListPasswordJobs(ctx) },
		status: func(ctx context.Context, c api.JobsAPI, id string) (json.RawMessage, error) { return c.GetPasswordJob(ctx, id) },
		results: func(ctx context.Context, c api.JobsAPI, id string, page, pageSize int, breached, foundOnly bool) (json.RawMessage, error) {
			return c.GetPasswordJobResults(ctx, id, page, pageSize, breached)
		},
		download: func(ctx context.Context, c api.JobsAPI, id string, breached, foundOnly bool) ([]byte, error) {
			return c.DownloadPasswordJob(ctx, id, breached)
		},
		cancel:        func(ctx context.Context, c api.JobsAPI, id string) (json.RawMessage, error) { return c.CancelPasswordJob(ctx, id) },
		retry:         nil,
		printJob:      printPasswordJob,
		listColumns:   []string{"ID", "Status", "Total", "Processed", "Breached", "Credits"},
		listRow:       passwordListRow,
		resultsKey:    "items",
		resultColumns: []string{"Line", "Prefix", "Breached", "Count"},
		resultRow:     passwordResultRow,
	},
}

// createIdentityJob reads emails from a file arg or STDIN, then submits the job.
func createIdentityJob(cmd *cobra.Command, args []string, c api.JobsAPI) (json.RawMessage, error) {
	path := ""
	if len(args) == 1 {
		path = args[0]
	}
	name, emails, _, err := loadEmails(cmd, path)
	if err != nil {
		return nil, err
	}
	fileName, _ := cmd.Flags().GetString("file-name")
	if fileName == "" {
		fileName = name
	}
	spinner := startSpinner("Creating identity job...")
	data, err := c.CreateIdentityJob(cmd.Context(), emails, fileName)
	stopSpinner(spinner)
	return data, err
}

// createPasswordJob gathers SHA-1 hashes (never plaintext) and submits the job.
func createPasswordJob(cmd *cobra.Command, args []string, c api.JobsAPI) (json.RawMessage, error) {
	sha1s, err := gatherPasswordSHA1s(cmd)
	if err != nil {
		return nil, err
	}
	if len(sha1s) == 0 {
		return nil, friendlyFormatError(cmd, "provide hashes via --sha1s/--sha1-file or --password-file")
	}
	fileName, _ := cmd.Flags().GetString("file-name")
	spinner := startSpinner("Creating password job...")
	data, err := c.CreatePasswordJob(cmd.Context(), sha1s, fileName)
	stopSpinner(spinner)
	return data, err
}
