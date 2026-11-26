package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/version"
	"golang.org/x/term"
)

// The "new version available" notice
//
// At most once a day the CLI asks GitHub for the latest release, in the
// background while the command runs, and caches the answer in
// ~/.encrata/update-check.json. When the command finishes, a newer version is
// announced with one line on STDERR. It never runs in CI, when STDERR is not a
// terminal, with --json or --quiet, or when ENCRATA_NO_UPDATE_NOTIFIER is set,
// and it never delays a command by more than noticeWait.

const (
	envNoUpdateNotifier = "ENCRATA_NO_UPDATE_NOTIFIER"
	updateCheckInterval = 24 * time.Hour
	releaseCheckTimeout = 3 * time.Second
	// noticeWait is the longest a finished command waits for a check still in
	// flight (only on the one run a day that checks, and only on a slow network).
	noticeWait = 800 * time.Millisecond
)

// noNoticeCommands never show the notice: they report versions themselves, or
// their output is machine-read.
var noNoticeCommands = map[string]bool{
	"update": true, "version": true, "doctor": true, "completion": true,
	"help": true, "__complete": true, "__completeNoDesc": true,
}

type updateCache struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

func updateCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".encrata", "update-check.json")
}

func loadUpdateCache() (updateCache, bool) {
	var c updateCache
	path := updateCachePath()
	if path == "" {
		return c, false
	}
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &c) != nil || c.Latest == "" {
		return updateCache{}, false
	}
	return c, true
}

// saveUpdateCache records the latest release seen. Best-effort.
func saveUpdateCache(latest string) {
	path := updateCachePath()
	if path == "" || latest == "" {
		return
	}
	data, err := json.Marshal(updateCache{CheckedAt: time.Now().UTC(), Latest: latest})
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, data, 0o600) != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// updateNotice carries one run's check. latest is written before done closes.
type updateNotice struct {
	current string
	latest  string
	done    chan struct{}
}

// updateNoticeEnabled decides, before the command runs, whether this run may
// show the notice.
func updateNoticeEnabled(args []string, getenv func(string) string, stderrTTY bool, current string) bool {
	if getenv(envNoUpdateNotifier) != "" || getenv("CI") != "" || !stderrTTY {
		return false
	}
	if _, ok := parseVersion(current); !ok { // "dev" and local builds
		return false
	}
	// The first word catches commands cobra adds lazily (completion, help);
	// Find resolves the rest, including any aliases.
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			if noNoticeCommands[a] {
				return false
			}
			break
		}
	}
	if c, _, err := rootCmd.Find(args); err == nil && c != nil {
		for p := c; p != nil; p = p.Parent() {
			if noNoticeCommands[p.Name()] {
				return false
			}
		}
	}
	return true
}

// startUpdateNotice begins this run's check, or returns nil when the notice is
// off. A fresh cache answers at once; a stale one is refreshed in the background.
func startUpdateNotice(args []string) *updateNotice {
	current := version.Version
	if !updateNoticeEnabled(args, os.Getenv, term.IsTerminal(int(os.Stderr.Fd())), current) {
		return nil
	}
	n := &updateNotice{current: current, done: make(chan struct{})}
	if c, ok := loadUpdateCache(); ok && time.Since(c.CheckedAt) < updateCheckInterval {
		n.latest = c.Latest
		close(n.done)
		return n
	}
	go func() {
		defer close(n.done)
		ctx, cancel := context.WithTimeout(context.Background(), releaseCheckTimeout)
		defer cancel()
		if latest, err := latestRelease(ctx); err == nil {
			saveUpdateCache(latest)
			n.latest = latest
		}
	}()
	return n
}

// print shows the notice after the command, when a newer version is known. Safe
// on a nil notice.
func (n *updateNotice) print(ctx context.Context) {
	if n == nil || ctx.Err() != nil || jsonOutput || output.Quiet() {
		return
	}
	select {
	case <-n.done:
	case <-time.After(noticeWait):
		return
	}
	if line, ok := updateNoticeLine(n.current, n.latest); ok {
		fmt.Fprintln(os.Stderr, line)
	}
}

// updateNoticeLine returns the notice text when latest is newer than current.
func updateNoticeLine(current, latest string) (string, bool) {
	if cmp, ok := compareVersions(latest, current); !ok || cmp <= 0 {
		return "", false
	}
	return "\n" + strings.Join([]string{
		"  " + output.Warn.Sprint("A new version of encrata is available:") + fmt.Sprintf(" v%s → v%s", current, latest),
		"  " + output.Dim.Sprint("Run: encrata update   (hide this: "+envNoUpdateNotifier+"=1)"),
	}, "\n"), true
}
