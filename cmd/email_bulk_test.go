package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Encratahq/cli/internal/api"
	"github.com/spf13/cobra"
)

// validityBulkFake records the validity jobs the CLI creates. Creating the job
// fails on purpose so the test stops before polling and exporting.
type validityBulkFake struct {
	api.API
	jobFiles []string
}

var errStopAfterCreate = errors.New("stop after create")

func (f *validityBulkFake) CreateValidityJobFile(_ context.Context, fileName string, _ []byte) (json.RawMessage, error) {
	f.jobFiles = append(f.jobFiles, fileName)
	return nil, errStopAfterCreate
}

func newValidityBulkTestCmd(t *testing.T, flags map[string]string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: "bulk"}
	registerBulkFlags(c, true)
	for k, v := range flags {
		if err := c.Flags().Set(k, v); err != nil {
			t.Fatalf("set --%s: %v", k, err)
		}
	}
	c.SetContext(context.Background())
	return c
}

// Every validity list, whatever its size and whatever the old --stream/--job
// flags say, is a job on POST /api/cli/jobs/bulk: the dashboard's path, so the
// same file gives the same counts, report and credits (ENC-2783).
func TestValidityBulkAlwaysRunsAsJob(t *testing.T) {
	for name, flags := range map[string]map[string]string{
		"default":       {},
		"legacy stream": {"stream": "true"},
		"legacy job":    {"job": "true"},
		"both legacy":   {"stream": "true", "job": "true"},
	} {
		t.Run(name, func(t *testing.T) {
			f := &validityBulkFake{}
			defer withFake(t, f)()
			path := writeEmailFile(t, "list.csv", 3)
			flags["out"] = path + ".out.csv"
			cmd := newValidityBulkTestCmd(t, flags)

			// A fake without BulkBreachesSearch / EmailValidity panics if the
			// CLI took any other route, so reaching the job error proves the path.
			if err := runEmailBulk(cmd, []string{path}); !errors.Is(err, errStopAfterCreate) {
				t.Fatalf("err = %v", err)
			}
			if len(f.jobFiles) != 1 || f.jobFiles[0] != "list.csv" {
				t.Fatalf("job files = %v", f.jobFiles)
			}
		})
	}
}
