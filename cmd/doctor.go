package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Encratahq/cli/internal/config"
	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/version"
	"github.com/spf13/cobra"
)

// errDoctorFailed makes `doctor` exit 1 when a check failed; the report is
// already printed.
var errDoctorFailed = errors.New("doctor found problems")

// Check outcomes, worst last.
const (
	checkOK   = "ok"
	checkInfo = "info"
	checkWarn = "warn"
	checkFail = "fail"
)

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	Hint   string `json:"hint,omitempty"`
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the CLI's install, PATH, configuration and API access",
	Long: `Diagnose the CLI: the version and whether a newer one exists, how it was
installed, which encrata your shell actually runs (and whether there are
several on PATH), whether it can update itself, which config and API key are
in use, and whether the API accepts that key.

Exits 1 when a check fails (warnings do not), so it works in scripts.

Examples:
  encrata doctor
  encrata doctor --json`,
	Args: cobra.NoArgs,
	RunE: runDoctor,
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	spinner := startSpinner("Running checks...")
	checks := doctorChecks(ctx)
	stopSpinner(spinner)

	failed := false
	for _, c := range checks {
		if c.Status == checkFail {
			failed = true
		}
	}

	if jsonMode() {
		data, _ := json.Marshal(map[string]interface{}{"ok": !failed, "checks": checks})
		output.JSON(data)
	} else {
		renderDoctor(checks)
	}
	if failed {
		return errDoctorFailed
	}
	return nil
}

func doctorChecks(ctx context.Context) []doctorCheck {
	var checks []doctorCheck
	checks = append(checks, versionCheck(ctx))

	exePath, err := runningExecutable()
	if err != nil {
		return append(checks, doctorCheck{Name: "Install", Status: checkFail, Detail: err.Error()})
	}
	method := detectInstallMethod(exePath)
	checks = append(checks, doctorCheck{Name: "Install", Status: checkInfo,
		Detail: fmt.Sprintf("%s (%s/%s) at %s", installMethodLabel(method), runtime.GOOS, runtime.GOARCH, exePath)})
	checks = append(checks, pathCheck(exePath, method, findOnPath(os.Getenv("PATH"))))
	if method == "direct" {
		checks = append(checks, writableCheck(exePath))
	}

	checks = append(checks, configChecks()...)
	checks = append(checks, apiCheck(ctx))
	return checks
}

func versionCheck(ctx context.Context) doctorCheck {
	c := doctorCheck{Name: "Version", Detail: fmt.Sprintf("v%s (commit %s, built %s)", version.Version, version.Commit, version.Date)}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	latest, err := latestRelease(cctx)
	if err != nil {
		c.Status = checkWarn
		c.Hint = "Could not check for a newer version: " + err.Error()
		return c
	}
	saveUpdateCache(latest)
	cmp, ok := compareVersions(version.Version, latest)
	switch {
	case !ok:
		c.Status = checkInfo
		c.Detail += fmt.Sprintf("; latest release is v%s", latest)
	case cmp < 0:
		c.Status = checkWarn
		c.Detail += fmt.Sprintf("; v%s is available", latest)
		c.Hint = "Run: encrata update"
	default:
		c.Status = checkOK
		c.Detail += "; up to date"
	}
	return c
}

func installMethodLabel(method string) string {
	switch method {
	case "npm":
		return "npm"
	case "brew":
		return "Homebrew"
	default:
		return "direct install"
	}
}

// findOnPath lists every encrata executable on PATH, in PATH order, at most one
// per directory.
func findOnPath(pathEnv string) []string {
	names := []string{"encrata"}
	if runtime.GOOS == "windows" {
		names = []string{"encrata.exe", "encrata.cmd", "encrata.bat"}
	}
	seen := map[string]bool{}
	var found []string
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		key := filepath.Clean(dir)
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		for _, n := range names {
			p := filepath.Join(dir, n)
			if info, err := os.Stat(p); err == nil && !info.IsDir() && (runtime.GOOS == "windows" || info.Mode()&0o111 != 0) {
				found = append(found, p)
				break
			}
		}
	}
	return found
}

// pathCheck reports which encrata the shell runs and flags duplicate installs,
// the usual cause of "I updated but nothing changed".
func pathCheck(exePath, method string, onPath []string) doctorCheck {
	c := doctorCheck{Name: "PATH"}
	if len(onPath) == 0 {
		c.Status = checkWarn
		c.Detail = "encrata is not on PATH"
		c.Hint = "Add " + filepath.Dir(exePath) + " to PATH, or reinstall."
		return c
	}
	if len(onPath) > 1 {
		c.Status = checkWarn
		c.Detail = fmt.Sprintf("%d copies on PATH; your shell runs the first: %s", len(onPath), strings.Join(onPath, ", "))
		c.Hint = "Remove the installs you don't use, so `encrata update` updates the one you run."
		return c
	}
	c.Status = checkOK
	c.Detail = onPath[0]
	// npm puts a launcher on PATH, not the binary, so only a direct or brew
	// install can be compared path-for-path.
	if method != "npm" && !sameFile(onPath[0], exePath) {
		c.Status = checkWarn
		c.Detail = fmt.Sprintf("the encrata on PATH (%s) is not this binary (%s)", onPath[0], exePath)
		c.Hint = "Run the one on PATH, or remove the other install."
	}
	return c
}

func sameFile(a, b string) bool {
	if ra, err := filepath.EvalSymlinks(a); err == nil {
		a = ra
	}
	if rb, err := filepath.EvalSymlinks(b); err == nil {
		b = rb
	}
	ia, errA := os.Stat(a)
	ib, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return os.SameFile(ia, ib)
}

// writableCheck reports whether `encrata update` can replace this binary, and
// whether a rollback copy is kept.
func writableCheck(exePath string) doctorCheck {
	c := doctorCheck{Name: "Self-update"}
	dir := filepath.Dir(exePath)
	tmp, err := os.CreateTemp(dir, updateTempPrefix+"probe-*")
	if err != nil {
		c.Status = checkWarn
		c.Detail = "cannot write to " + dir
		c.Hint = strings.TrimPrefix(permissionHint(dir, err).Error(), "no permission to write to "+dir+": ")
		return c
	}
	tmp.Close()
	os.Remove(tmp.Name())
	c.Status = checkOK
	c.Detail = "can update in place"
	if _, err := os.Stat(previousPath(exePath)); err == nil {
		c.Detail += "; previous version kept (encrata update --rollback)"
	}
	if n := len(updateLeftovers(exePath)); n > 0 {
		c.Detail += fmt.Sprintf("; %d leftover update file(s), removed on a later run", n)
	}
	return c
}

func configChecks() []doctorCheck {
	file := config.FileUsed()
	cfgCheck := doctorCheck{Name: "Config", Status: checkInfo, Detail: "no config file (defaults and environment only)"}
	if file != "" {
		cfgCheck.Detail = file
	}

	key := doctorCheck{Name: "API key", Status: checkOK}
	switch {
	case rootCmd.PersistentFlags().Changed("api-key"):
		key.Detail = "from --api-key"
	case os.Getenv(config.EnvAPIKey) != "":
		key.Detail = "from " + config.EnvAPIKey
	case app.cfg != nil && app.cfg.APIKey != "":
		key.Detail = "from the config file"
	default:
		key.Status = checkFail
		key.Detail = "none set"
		key.Hint = "Run: encrata login"
	}
	if app.cfg != nil && app.cfg.APIKey != "" {
		key.Detail += " (" + maskKey(app.cfg.APIKey) + ")"
	}

	base := doctorCheck{Name: "API URL", Status: checkInfo}
	if app.cfg != nil {
		base.Detail = app.cfg.BaseURL
		if strings.TrimRight(app.cfg.BaseURL, "/") != config.DefaultBaseURL {
			base.Detail += " (custom; the default is " + config.DefaultBaseURL + ")"
		}
	}
	return []doctorCheck{cfgCheck, key, base}
}

// maskKey shows enough of a key to recognise it and no more.
func maskKey(k string) string {
	if len(k) <= 12 {
		return strings.Repeat("•", len(k))
	}
	return k[:8] + "…" + k[len(k)-4:]
}

func apiCheck(ctx context.Context) doctorCheck {
	c := doctorCheck{Name: "API access"}
	if app.cfg == nil || app.cfg.APIKey == "" {
		c.Status = checkInfo
		c.Detail = "skipped: no API key"
		return c
	}
	client, err := newClient()
	if err != nil {
		c.Status = checkFail
		c.Detail = err.Error()
		return c
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	data, err := client.Me(cctx)
	if err != nil {
		c.Status = checkFail
		c.Detail = err.Error()
		c.Hint = "Check the key with `encrata login`, and the URL with `encrata config show`."
		return c
	}
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	c.Status = checkOK
	c.Detail = "authenticated"
	if email := field(m, "email", "result.email"); email != "" {
		c.Detail += " as " + email
	}
	return c
}

func renderDoctor(checks []doctorCheck) {
	output.Header("Encrata Doctor")
	width := 0
	for _, c := range checks {
		if len(c.Name) > width {
			width = len(c.Name)
		}
	}
	var warns, fails int
	for _, c := range checks {
		mark := output.Dim.Sprint("·")
		switch c.Status {
		case checkOK:
			mark = output.Success.Sprint("✓")
		case checkWarn:
			mark = output.Warn.Sprint("!")
			warns++
		case checkFail:
			mark = output.Err.Sprint("✗")
			fails++
		}
		fmt.Printf("  %s %-*s  %s\n", mark, width, c.Name, c.Detail)
		if c.Hint != "" {
			fmt.Printf("    %*s %s\n", width, "", output.Dim.Sprint(c.Hint))
		}
	}
	fmt.Println()
	switch {
	case fails > 0:
		output.Error(fmt.Sprintf("%d %s, %d %s.", fails, plural(fails, "problem", "problems"), warns, plural(warns, "warning", "warnings")))
	case warns > 0:
		output.Info(fmt.Sprintf("No problems, %d %s.", warns, plural(warns, "warning", "warnings")))
	default:
		output.SuccessMsg("Everything looks good.")
	}
}
