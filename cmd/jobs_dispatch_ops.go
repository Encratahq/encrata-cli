package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// ── results ─────────────────────────────────────────────────────────────────

func resultsNonValidityJob(cmd *cobra.Command, args []string, jt string) error {
	s := strategyFor(jt)
	client, err := newClient()
	if err != nil {
		return err
	}
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	breached, _ := cmd.Flags().GetBool("breached")
	foundOnly, _ := cmd.Flags().GetBool("found-only")
	_, _, onlyFound, onlyBreached := resolveOnlyFilters(cmd)
	breached = breached || onlyBreached
	foundOnly = foundOnly || onlyFound

	spinner := startSpinner("Loading results...")
	data, err := s.results(cmd.Context(), client, args[0], page, pageSize, breached, foundOnly)
	stopSpinner(spinner)
	if err != nil {
		return jobsError(err)
	}
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	items := unwrapArray(data, s.resultsKey)
	output.Header(fmt.Sprintf("Results: %d", len(items)))
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			rows = append(rows, s.resultRow(m))
		}
	}
	output.Table(s.resultColumns, rows)
	return nil
}

// ── download ────────────────────────────────────────────────────────────────

func downloadNonValidityJob(cmd *cobra.Command, args []string, jt string) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	out, _ := cmd.Flags().GetString("out")
	format, ferr := resolveExportFormat(flagString(cmd, "format"), out)
	if ferr != nil {
		return friendlyFormatError(cmd, ferr.Error())
	}
	if out == "" && !jsonMode() && (format == "csv" || format == "xlsx") {
		out = defaultResultPath(args[0], format)
	}
	// Server-built cached export (streamed from S3), replacing the old sync
	// streaming download that pulled every row into the CLI.
	return runJobExport(cmd.Context(), client, jt, args[0], out, format, nonValidityExportFilter(cmd, jt))
}

// ── cancel ──────────────────────────────────────────────────────────────────

func cancelNonValidityJob(cmd *cobra.Command, args []string, jt string) error {
	s := strategyFor(jt)
	client, err := newClient()
	if err != nil {
		return err
	}
	spinner := startSpinner("Cancelling job...")
	data, err := s.cancel(cmd.Context(), client, args[0])
	stopSpinner(spinner)
	if err != nil {
		return jobsError(err)
	}
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	output.SuccessMsg("Job cancelled: " + args[0])
	return nil
}

// ── retry ───────────────────────────────────────────────────────────────────

func retryNonValidityJob(cmd *cobra.Command, args []string, jt string) error {
	s := strategyFor(jt)
	if s.retry == nil {
		return friendlyFormatError(cmd, "password jobs cannot be retried")
	}
	client, err := newClient()
	if err != nil {
		return err
	}
	spinner := startSpinner("Retrying job...")
	data, err := s.retry(cmd.Context(), client, args[0])
	stopSpinner(spinner)
	if err != nil {
		return jobsError(err)
	}
	if jsonMode() {
		output.JSON(data)
		return nil
	}
	requeued := 0
	var m map[string]interface{}
	if json.Unmarshal(data, &m) == nil {
		requeued = getInt(m, "requeued")
	}
	output.SuccessMsg(fmt.Sprintf("Requeued %d %s", requeued, plural(requeued, "chunk", "chunks")))

	statusData, err := s.status(cmd.Context(), client, args[0])
	if err != nil {
		return nil
	}
	var job map[string]interface{}
	if !decode(statusData, &job) {
		return nil
	}
	output.Header(s.title + " Job: " + args[0])
	s.printJob(job)
	return nil
}
