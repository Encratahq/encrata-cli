package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestJobStrategiesComplete verifies every non-validity job type declares a
// complete strategy, so adding a type stays a one-entry change (Open/Closed).
func TestJobStrategiesComplete(t *testing.T) {
	for _, jt := range []string{"identity", "password"} {
		s := strategyFor(jt)
		if s.title == "" {
			t.Errorf("%s: missing title", jt)
		}
		if s.create == nil || s.list == nil || s.status == nil ||
			s.results == nil || s.download == nil || s.cancel == nil {
			t.Errorf("%s: incomplete verb set", jt)
		}
		if s.printJob == nil || s.listRow == nil || s.resultRow == nil {
			t.Errorf("%s: missing renderers", jt)
		}
		if len(s.listColumns) == 0 || len(s.resultColumns) == 0 || s.resultsKey == "" {
			t.Errorf("%s: missing table metadata", jt)
		}
	}

	if strategyFor("password").retry != nil {
		t.Error("password jobs must not support retry")
	}
	if strategyFor("identity").retry == nil {
		t.Error("identity jobs must support retry")
	}
}

// TestJobType validates the --type flag parsing on the unified jobs surface.
func TestJobType(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", "validity", false},
		{"validity", "validity", false},
		{"identity", "identity", false},
		{"password", "password", false},
		{"bogus", "", true},
	}
	for _, c := range cases {
		cmd := &cobra.Command{}
		cmd.Flags().String("type", "validity", "")
		if err := cmd.Flags().Set("type", c.in); err != nil {
			t.Fatalf("set --type=%q: %v", c.in, err)
		}
		got, err := jobType(cmd)
		if c.wantErr {
			if err == nil {
				t.Errorf("--type=%q: expected error, got none", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("--type=%q: unexpected error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("--type=%q: got %q, want %q", c.in, got, c.want)
		}
	}
}
