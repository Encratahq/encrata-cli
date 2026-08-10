package cmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/config"
	"github.com/spf13/cobra"
)

// fakeAPI embeds api.API so it satisfies the full interface; only the methods a
// test exercises are overridden. Any un-overridden call panics, surfacing
// unexpected network use.
type fakeAPI struct {
	api.API
	listIdentityCalled bool
	cancelPasswordID   string
}

func (f *fakeAPI) ListIdentityJobs(context.Context) (json.RawMessage, error) {
	f.listIdentityCalled = true
	return json.RawMessage(`{"jobs":[]}`), nil
}

func (f *fakeAPI) CancelPasswordJob(_ context.Context, id string) (json.RawMessage, error) {
	f.cancelPasswordID = id
	return json.RawMessage(`{"status":"cancelled"}`), nil
}

// withFake swaps the injectable dependency seam for a fake and restores it.
func withFake(t *testing.T, f api.API) func() {
	t.Helper()
	prev := app
	app = &appDeps{
		cfg:    &config.Config{Output: "table", BaseURL: "http://test", APIKey: "k"},
		newAPI: func(*config.Config) (api.API, error) { return f, nil },
	}
	return func() { app = prev }
}

func newJobsTestCmd() *cobra.Command {
	c := &cobra.Command{}
	c.Flags().Int("page", 1, "")
	c.Flags().Int("page-size", 50, "")
	c.Flags().Bool("breached", false, "")
	c.Flags().Bool("found-only", false, "")
	c.Flags().String("out", "", "")
	c.SetContext(context.Background())
	return c
}

func TestNewClientReturnsInjectedFake(t *testing.T) {
	f := &fakeAPI{}
	defer withFake(t, f)()
	got, err := newClient()
	if err != nil {
		t.Fatalf("newClient: %v", err)
	}
	if got != api.API(f) {
		t.Fatal("newClient did not return the injected fake")
	}
}

func TestListNonValidityJobsUsesInjectedClient(t *testing.T) {
	f := &fakeAPI{}
	defer withFake(t, f)()
	if err := listNonValidityJobs(newJobsTestCmd(), "identity"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.listIdentityCalled {
		t.Fatal("expected ListIdentityJobs to be called on the injected fake")
	}
}

func TestCancelNonValidityJobTargetsCorrectID(t *testing.T) {
	f := &fakeAPI{}
	defer withFake(t, f)()
	if err := cancelNonValidityJob(newJobsTestCmd(), []string{"job-123"}, "password"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.cancelPasswordID != "job-123" {
		t.Fatalf("expected cancel to target job-123, got %q", f.cancelPasswordID)
	}
}
