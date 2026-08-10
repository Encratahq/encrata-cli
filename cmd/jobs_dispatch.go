package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// jobType reads and validates the --type flag on the unified `jobs` commands.
// Defaults to validity so existing (validity-only) invocations are unchanged.
func jobType(cmd *cobra.Command) (string, error) {
	t, _ := cmd.Flags().GetString("type")
	switch t {
	case "", "validity":
		return "validity", nil
	case "identity", "password":
		return t, nil
	default:
		return "", friendlyFormatError(cmd, "--type must be validity, identity, or password")
	}
}

// ── create ──────────────────────────────────────────────────────────────────

func createNonValidityJob(cmd *cobra.Command, args []string, jt string) error {
	s := strategyFor(jt)
	client, err := newClient()
	if err != nil {
		return err
	}
	data, err := s.create(cmd, args, client)
	if err != nil {
		return jobsError(err)
	}
	return renderCreatedJob(s, data)
}

func renderCreatedJob(s jobStrategy, data json.RawMessage) error {
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	var m map[string]interface{}
	if !decode(data, &m) {
		return nil
	}
	output.Header(s.title + " Job Created")
	s.printJob(m)
	return nil
}

// ── list ────────────────────────────────────────────────────────────────────

func listNonValidityJobs(cmd *cobra.Command, jt string) error {
	s := strategyFor(jt)
	client, err := newClient()
	if err != nil {
		return err
	}
	spinner := startSpinner("Loading jobs...")
	data, err := s.list(cmd.Context(), client)
	stopSpinner(spinner)
	if err != nil {
		return jobsError(err)
	}
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	jobs := unwrapArray(data, "jobs")
	output.Header(fmt.Sprintf("%s Jobs: %d", s.title, len(jobs)))
	rows := make([][]string, 0, len(jobs))
	for _, item := range jobs {
		if m, ok := item.(map[string]interface{}); ok {
			rows = append(rows, s.listRow(m))
		}
	}
	output.Table(s.listColumns, rows)
	return nil
}

// ── status ──────────────────────────────────────────────────────────────────

func statusNonValidityJob(cmd *cobra.Command, args []string, jt string) error {
	s := strategyFor(jt)
	client, err := newClient()
	if err != nil {
		return err
	}
	spinner := startSpinner("Loading job...")
	data, err := s.status(cmd.Context(), client, args[0])
	stopSpinner(spinner)
	if err != nil {
		return jobsError(err)
	}
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	var m map[string]interface{}
	if !decode(data, &m) {
		return nil
	}
	output.Header(s.title + " Job: " + args[0])
	s.printJob(m)
	return nil
}

// jobsError prints an API error and returns it so the command exits non-zero.
func jobsError(err error) error {
	output.Error(err.Error())
	return err
}
