package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Encratahq/cli/internal/config"
	"github.com/Encratahq/cli/internal/notify"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

// configNotificationsCmd manages the local desktop-notification preference.
// It lives in ~/.encrata/config.yaml on this machine only — deliberately not a
// dashboard/account setting, so each machine opts in on its own.
var configNotificationsCmd = &cobra.Command{
	Use:   "notifications [on|off|test]",
	Short: "Desktop notifications when a bulk job finishes (local, off by default)",
	Long: `Show a desktop notification when a bulk job you are waiting on finishes,
with a "View results" link to the dashboard. Notifications are off until you
turn them on, and the setting is stored on this machine only.

Only finished jobs notify (never progress), a job you waited on for under 30s
in the foreground does not notify, and several jobs finishing together while
following with 'encrata jobs watch' are grouped into one notification.

  on     turn notifications on (optionally limit with --on)
  off    turn notifications off
  test   send a sample notification
  (none) show the current setting

Override per run with --notify or --no-notify.`,
	Example: `  encrata config notifications on
  encrata config notifications on --on completed,failed
  encrata config notifications test
  encrata config notifications off`,
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"on", "off", "test"},
	RunE: func(cmd *cobra.Command, args []string) error {
		action := ""
		if len(args) == 1 {
			action = strings.ToLower(strings.TrimSpace(args[0]))
		}
		on, _ := cmd.Flags().GetStringSlice("on")
		if cmd.Flags().Changed("on") && action != "on" {
			return friendlyFormatError(cmd, "--on is only used with: encrata config notifications on")
		}
		prefs := &app.cfg.Notifications
		switch action {
		case "":
			printNotifyPrefs(*prefs)
			return nil
		case "test":
			return notify.Send(notify.Message{
				Title: "Encrata notifications are working",
				Body:  "You'll see this when a bulk job you're waiting on finishes.",
				URL:   strings.TrimRight(app.cfg.AppURL, "/"),
			})
		case "on":
			events, err := parseNotifyEvents(on)
			if err != nil {
				return friendlyFormatError(cmd, err.Error())
			}
			prefs.Enabled, prefs.On = true, events
		case "off":
			prefs.Enabled = false
		default:
			return friendlyFormatError(cmd, fmt.Sprintf("unknown option %q; use on, off, or test", args[0]))
		}
		if err := config.Save(app.cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		printNotifyPrefs(*prefs)
		return nil
	},
}

func init() {
	configNotificationsCmd.Flags().StringSlice("on", nil, "Outcomes to notify on: completed, failed, cancelled (default all)")
	configCmd.AddCommand(configNotificationsCmd)
}

// parseNotifyEvents validates --on, defaulting to every outcome.
func parseNotifyEvents(in []string) ([]string, error) {
	if len(in) == 0 {
		return slices.Clone(config.NotifyEvents), nil
	}
	out := []string{}
	for _, e := range in {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "canceled" {
			e = "cancelled"
		}
		if !slices.Contains(config.NotifyEvents, e) {
			return nil, fmt.Errorf("unknown outcome %q; use completed, failed, or cancelled", e)
		}
		if !slices.Contains(out, e) {
			out = append(out, e)
		}
	}
	return out, nil
}

// notifySummary renders the preference for `config show` and this command.
func notifySummary(p config.NotifyPrefs) string {
	if !p.Enabled {
		return "off"
	}
	on := p.On
	if len(on) == 0 {
		on = config.NotifyEvents
	}
	return "on (" + strings.Join(on, ", ") + ")"
}

func printNotifyPrefs(p config.NotifyPrefs) {
	state := output.Dim.Sprint(notifySummary(p))
	if p.Enabled {
		state = output.Success.Sprint(notifySummary(p))
	}
	fmt.Printf("  Desktop notifications: %s\n", state)
}
