package cmd

import (
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var monitorsCmd = &cobra.Command{
	Use:   "monitors",
	Short: "Manage monitors",
}

var monitorsLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all monitors",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.ListMonitors(cmd.Context())
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		monitors := unwrapArray(data, "monitors")
		output.Header("Monitors")
		if len(monitors) == 0 {
			output.Dim.Println("  No monitors found")
			return nil
		}
		rows := make([][]string, 0, len(monitors))
		for _, item := range monitors {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "name"),
				getStr(m, "status"),
				getStr(m, "frequency"),
				fmt.Sprintf("%d", getInt(m, "email_count")),
			})
		}
		output.Table([]string{"ID", "Name", "Status", "Frequency", "Emails"}, rows)
		return nil
	},
}

var monitorsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a monitor",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		emails, _ := cmd.Flags().GetStringSlice("emails")
		listID, _ := cmd.Flags().GetString("list-id")
		frequency, _ := cmd.Flags().GetString("frequency")
		changeDetection, _ := cmd.Flags().GetString("change-detection")

		req := &api.MonitorCreateRequest{
			Name:            args[0],
			Frequency:       frequency,
			ChangeDetection: changeDetection,
			Emails:          emails,
		}
		if listID != "" {
			req.DataSourceType = "list"
			req.DataSourceRef = listID
		}

		data, err := client.CreateMonitor(cmd.Context(), req)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var m map[string]interface{}
		if decode(data, &m) {
			output.SuccessMsg("Monitor created")
			output.KV("ID", getStr(m, "id"), "Name", getStr(m, "name"))
		}
		return nil
	},
}

var monitorsGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Show a monitor",
	Args:  cobra.ExactArgs(1),
	RunE:  simpleGet((*api.Client).GetMonitor, "Monitor"),
}

var monitorsRunCmd = &cobra.Command{
	Use:   "run [id]",
	Short: "Trigger a monitoring run",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := client.TriggerMonitorRun(cmd.Context(), args[0])
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		var m map[string]interface{}
		if decode(data, &m) {
			output.SuccessMsg("Run triggered")
			output.KV("Run ID", getStr(m, "run_id"), "Status", getStr(m, "status"))
		}
		return nil
	},
}

var monitorsRunsCmd = &cobra.Command{
	Use:   "runs [id]",
	Short: "List runs for a monitor",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		data, err := client.ListMonitorRuns(cmd.Context(), args[0], limit, offset)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		runs := unwrapArray(data, "runs")
		output.Header("Monitor Runs")
		if len(runs) == 0 {
			output.Dim.Println("  No runs found")
			return nil
		}
		rows := make([][]string, 0, len(runs))
		for _, item := range runs {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "status"),
				fmt.Sprintf("%d", getInt(m, "total_records")),
				fmt.Sprintf("%d", getInt(m, "changes_detected")),
				getStr(m, "completed_at"),
			})
		}
		output.Table([]string{"Run ID", "Status", "Records", "Changes", "Completed"}, rows)
		return nil
	},
}

var monitorsResultsCmd = &cobra.Command{
	Use:   "results [monitor-id] [run-id]",
	Short: "Show results for a monitoring run",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		changesOnly, _ := cmd.Flags().GetBool("changes-only")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		data, err := client.GetMonitorRunResults(cmd.Context(), args[0], args[1], changesOnly, limit, offset)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		output.JSON(data)
		return nil
	},
}

var monitorsAllRunsCmd = &cobra.Command{
	Use:   "all-runs",
	Short: "List runs across all monitors",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		data, err := client.ListAllMonitorRuns(cmd.Context(), limit, offset)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if jsonMode() {
			output.JSON(data)
			return nil
		}
		runs := unwrapArray(data, "runs")
		output.Header("Monitor Runs (all)")
		if len(runs) == 0 {
			output.Dim.Println("  No runs found")
			return nil
		}
		rows := make([][]string, 0, len(runs))
		for _, item := range runs {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			rows = append(rows, []string{
				getStr(m, "id"),
				getStr(m, "monitor_name"),
				getStr(m, "status"),
				fmt.Sprintf("%d", getInt(m, "changes_detected")),
				getStr(m, "completed_at"),
			})
		}
		output.Table([]string{"Run ID", "Monitor", "Status", "Changes", "Completed"}, rows)
		return nil
	},
}

var monitorsAllResultsCmd = &cobra.Command{
	Use:   "all-results",
	Short: "List results across all monitors",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		changesOnly, _ := cmd.Flags().GetBool("changes-only")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		data, err := client.ListAllMonitorResults(cmd.Context(), changesOnly, limit, offset)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		output.JSON(data)
		return nil
	},
}

func init() {
	monitorsCreateCmd.Flags().StringSlice("emails", nil, "Emails to monitor")
	monitorsCreateCmd.Flags().String("list-id", "", "Contact list ID to use as the data source")
	monitorsCreateCmd.Flags().String("frequency", "monthly", "Frequency (weekly, biweekly, monthly, quarterly)")
	monitorsCreateCmd.Flags().String("change-detection", "diff_only", "Change detection (diff_only, full_refresh)")

	monitorsRunsCmd.Flags().Int("limit", 20, "Results per page")
	monitorsRunsCmd.Flags().Int("offset", 0, "Result offset")

	monitorsResultsCmd.Flags().Bool("changes-only", false, "Only show records with changes")
	monitorsResultsCmd.Flags().Int("limit", 100, "Results per page")
	monitorsResultsCmd.Flags().Int("offset", 0, "Result offset")

	monitorsAllRunsCmd.Flags().Int("limit", 20, "Results per page")
	monitorsAllRunsCmd.Flags().Int("offset", 0, "Result offset")

	monitorsAllResultsCmd.Flags().Bool("changes-only", false, "Only show records with changes")
	monitorsAllResultsCmd.Flags().Int("limit", 100, "Results per page")
	monitorsAllResultsCmd.Flags().Int("offset", 0, "Result offset")

	monitorsCmd.AddCommand(monitorsLsCmd, monitorsCreateCmd, monitorsGetCmd, monitorsRunCmd, monitorsRunsCmd, monitorsResultsCmd, monitorsAllRunsCmd, monitorsAllResultsCmd)
}
