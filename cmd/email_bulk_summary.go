package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/output"
)

// normStatus normalizes a validity status for bucketing.
func normStatus(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), "_", "-")
}

// rawResultsArray assembles the raw per-row payloads into a single JSON array,
// preserving the exact server bytes for --json parity with the MCP server.
func rawResultsArray(rows []json.RawMessage) json.RawMessage {
	if len(rows) == 0 {
		return json.RawMessage("[]")
	}
	var buf strings.Builder
	buf.WriteByte('[')
	for i, row := range rows {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(row)
	}
	buf.WriteByte(']')
	return json.RawMessage(buf.String())
}

// bulkCounts tallies statuses and credits across a result set.
func bulkCounts(results []map[string]interface{}) (total, valid, invalid, catchall, risky, credits int) {
	total = len(results)
	for _, r := range results {
		switch normStatus(field(r, "validity", "status")) {
		case "valid", "deliverable":
			valid++
		case "invalid", "undeliverable":
			invalid++
		case "catch-all", "catchall", "accept-all":
			catchall++
		case "risky", "risk":
			risky++
		}
		credits += intOf(field(r, "credits", "credits_used"))
	}
	if credits == 0 {
		credits = valid
	}
	return
}

// sumCredits totals the credits charged across a bulk result set.
func sumCredits(results []map[string]interface{}) int {
	total := 0
	for _, r := range results {
		total += intOf(field(r, "credits", "credits_used", "credits_charged"))
	}
	return total
}

// printBulkSummaryLine prints the colored aggregate summary line.
func printBulkSummaryLine(results []map[string]interface{}) {
	total, valid, invalid, catchall, risky, credits := bulkCounts(results)
	fmt.Printf("  Total %d · Valid %s · Invalid %s · Catch-all %s · Risky %s · Credits %d\n",
		total,
		output.Success.Sprintf("%d", valid),
		output.Err.Sprintf("%d", invalid),
		output.Warn.Sprintf("%d", catchall),
		output.Brand.Sprintf("%d", risky),
		credits,
	)
}

// printResultsTable renders Email | Status | Reason plus any --fields columns.
func printResultsTable(results []map[string]interface{}, fields []string) {
	if len(results) == 0 {
		return
	}
	headers := append([]string{"Email", "Status", "Reason"}, fields...)
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		row := []string{
			firstNonEmpty(field(r, "email"), "—"),
			firstNonEmpty(field(r, "validity", "status"), "—"),
			firstNonEmpty(field(r, "reason", "message"), "—"),
		}
		for _, f := range fields {
			row = append(row, cellForField(r, f))
		}
		rows = append(rows, row)
	}
	output.Table(headers, rows)
	fmt.Println()
}

// cellForField renders an arbitrary validity-schema field for table columns.
func cellForField(r map[string]interface{}, f string) string {
	if v, ok := lookupRaw(r, f); ok {
		if arr, ok := v.([]interface{}); ok {
			return firstNonEmpty(joinInterfaces(arr), "—")
		}
	}
	return firstNonEmpty(field(r, f), "—")
}
