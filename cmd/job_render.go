package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/output"
)

// printIdentityJob renders an identity job status block.
func printIdentityJob(m map[string]interface{}) {
	output.KV(
		"ID", getStr(m, "id"),
		"Status", coloredStatus(getStr(m, "status")),
		"Total", fmt.Sprintf("%d", getInt(m, "total_emails")),
		"Processed", fmt.Sprintf("%d", getInt(m, "processed_count")),
		"Found", fmt.Sprintf("%d", getInt(m, "found_count")),
		"Credits used", fmt.Sprintf("%d", getInt(m, "credits_used")),
		"Created", jobCreated(getStr(m, "created_at")),
	)
}

// printPasswordJob renders a password job status block.
func printPasswordJob(m map[string]interface{}) {
	output.KV(
		"ID", getStr(m, "id"),
		"Status", coloredStatus(getStr(m, "status")),
		"Total", fmt.Sprintf("%d", getInt(m, "total")),
		"Processed", fmt.Sprintf("%d", getInt(m, "processed_count")),
		"Breached", fmt.Sprintf("%d", getInt(m, "breached_count")),
		"Credits used", fmt.Sprintf("%d", getInt(m, "credits_used")),
		"Created", jobCreated(getStr(m, "created_at")),
	)
}

// Row builders map a job/result record to a table row. Kept per-type so the
// registry stays declarative and the dispatchers carry no rendering switches.

func identityListRow(m map[string]interface{}) []string {
	return []string{
		getStr(m, "id"), getStr(m, "status"),
		fmt.Sprintf("%d", getInt(m, "total_emails")),
		fmt.Sprintf("%d", getInt(m, "processed_count")),
		fmt.Sprintf("%d", getInt(m, "found_count")),
		fmt.Sprintf("%d", getInt(m, "credits_used")),
	}
}

func passwordListRow(m map[string]interface{}) []string {
	return []string{
		getStr(m, "id"), getStr(m, "status"),
		fmt.Sprintf("%d", getInt(m, "total")),
		fmt.Sprintf("%d", getInt(m, "processed_count")),
		fmt.Sprintf("%d", getInt(m, "breached_count")),
		fmt.Sprintf("%d", getInt(m, "credits_used")),
	}
}

func identityResultRow(m map[string]interface{}) []string {
	return []string{
		firstNonEmpty(getStr(m, "email"), "—"),
		yesNo(boolField(m, "found")),
		firstNonEmpty(getStr(m, "name"), "—"),
		firstNonEmpty(getStr(m, "company"), "—"),
		firstNonEmpty(getStr(m, "job_role"), "—"),
		firstNonEmpty(getStr(m, "location"), "—"),
	}
}

func passwordResultRow(m map[string]interface{}) []string {
	return []string{
		fmt.Sprintf("%d", getInt(m, "line_no")),
		firstNonEmpty(getStr(m, "prefix"), "—"),
		yesNo(boolField(m, "found")),
		fmt.Sprintf("%d", getInt(m, "count")),
	}
}
