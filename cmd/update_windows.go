//go:build windows

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Encratahq/cli/internal/output"
)

// createNewConsole opens the detached updater in its own window so the user can
// watch npm's progress after this process exits.
const createNewConsole = 0x00000010

// detachedNpmUpdate launches the npm update in a separate console that first
// waits for this process to exit, then installs the new version and removes the
// leftover ".encrata-cli-*" staging folders npm can't delete while the old
// binary is locked. Returns true when it has taken over and the caller should
// stop (and exit, releasing the lock).
func detachedNpmUpdate(exePath string) (bool, error) {
	pid := os.Getpid()
	modules := nodeModulesDir(exePath)

	var b strings.Builder
	fmt.Fprintf(&b, "try { Wait-Process -Id %d -Timeout 120 -ErrorAction SilentlyContinue } catch {}; ", pid)
	b.WriteString("npm install -g --no-fund --no-audit encrata-cli@latest; ")
	if modules != "" {
		// Delete the old staging copies now that the lock is gone.
		fmt.Fprintf(&b, "Get-ChildItem -LiteralPath '%s' -Filter '.encrata-cli-*' -Directory -ErrorAction SilentlyContinue | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue; ", strings.ReplaceAll(modules, "'", "''"))
	}
	b.WriteString("Write-Host ''; Write-Host 'Encrata CLI update complete - old version removed. You can close this window.'")

	c := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", b.String())
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewConsole}
	if err := c.Start(); err != nil {
		return false, err
	}
	// Release the started process so it keeps running after we exit.
	_ = c.Process.Release()

	output.Info("Finishing the update in a new window; it will replace and delete the old version once this process exits.")
	output.SuccessMsg("Update started.")
	return true, nil
}

// nodeModulesDir returns the node_modules directory that contains the running
// binary, where npm leaves its ".encrata-cli-*" staging folders.
func nodeModulesDir(exePath string) string {
	d := filepath.Dir(exePath)
	for {
		if strings.EqualFold(filepath.Base(d), "node_modules") {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}
