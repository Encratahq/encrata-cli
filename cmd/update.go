package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/version"
	"github.com/spf13/cobra"
)

// updateRepo is the GitHub repository that hosts the release archives.
const updateRepo = "Encratahq/encrata-cli"

// npmPackage is the published npm package name.
const npmPackage = "encrata-cli"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the CLI to the latest version",
	Long: `Check for a newer release and update in place.

Detects how the CLI was installed (npm, Homebrew, or the direct install
script) and uses the matching update method automatically.

A direct install is updated in place: the download is checked against the
release's checksums, the new binary must start before the update counts, and
the previous version is kept so --rollback can restore it.

Examples:
  encrata update            # update to the latest release
  encrata update --check    # only report whether an update is available
  encrata update --rollback # go back to the version before the last update`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if rb, _ := cmd.Flags().GetBool("rollback"); rb {
			return runRollback()
		}
		check, _ := cmd.Flags().GetBool("check")
		return runUpdate(cmd.Context(), check)
	},
}

func init() {
	updateCmd.Flags().Bool("check", false, "Only check whether a newer version is available")
	updateCmd.Flags().Bool("rollback", false, "Restore the version that was installed before the last update")
	updateCmd.MarkFlagsMutuallyExclusive("check", "rollback")
}

func runUpdate(ctx context.Context, checkOnly bool) error {
	current := version.Version
	output.Header("Update Encrata CLI")
	fmt.Printf("  Current version: v%s\n", current)

	spinner := startSpinner("Checking for updates...")
	latest, err := latestRelease(ctx)
	stopSpinner(spinner)
	if err != nil {
		output.Error("Could not check for updates: " + err.Error())
		return err
	}
	fmt.Printf("  Latest version:  v%s\n\n", latest)
	saveUpdateCache(latest) // keeps the update notice in step with this check

	if cmp, ok := compareVersions(current, latest); ok {
		switch {
		case cmp == 0:
			output.SuccessMsg("You're already on the latest version.")
			return nil
		case cmp > 0:
			output.SuccessMsg(fmt.Sprintf("v%s is newer than the latest release; nothing to update.", current))
			return nil
		}
	}
	// A "dev" or otherwise unparseable build always counts as older.

	if checkOnly {
		output.Info(fmt.Sprintf("v%s is available. Run: encrata update", latest))
		return nil
	}

	exePath, err := runningExecutable()
	if err != nil {
		return err
	}

	switch detectInstallMethod(exePath) {
	case "npm":
		// On Windows the running .exe is locked, so npm can't replace or delete
		// the old copy. Hand off to a detached updater that runs after we exit.
		if done, err := detachedNpmUpdate(exePath); err != nil {
			output.Error("Could not start background update: " + err.Error() + " - falling back to in-place update.")
		} else if done {
			return nil
		}
		return runManagerUpdate("npm", []string{"install", "-g", "--no-fund", "--no-audit", npmPackage + "@latest"})
	case "brew":
		return runManagerUpdate("brew", []string{"upgrade", "encrata"})
	default:
		if err := selfUpdate(ctx, exePath, current, latest); err != nil {
			output.Error(err.Error())
			return err
		}
		return nil
	}
}

func runRollback() error {
	exePath, err := runningExecutable()
	if err != nil {
		return err
	}
	switch detectInstallMethod(exePath) {
	case "npm":
		return fmt.Errorf("this CLI was installed with npm; install a specific version with: npm install -g %s@<version>", npmPackage)
	case "brew":
		return fmt.Errorf("this CLI was installed with Homebrew, which keeps only the current version; reinstall a specific version from https://github.com/%s/releases", updateRepo)
	}
	if err := rollbackBinary(exePath); err != nil {
		output.Error(err.Error())
		return err
	}
	now := "the previous version"
	if out, err := exec.Command(exePath, "version").Output(); err == nil {
		if line := strings.TrimSpace(strings.SplitN(stripANSI(string(out)), "\n", 2)[0]); line != "" {
			now = line
		}
	}
	output.SuccessMsg("Rolled back to " + now + ". Run `encrata update --rollback` again to undo.")
	return nil
}

// runningExecutable returns the resolved path of this binary.
func runningExecutable() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not locate the running binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}
	return exePath, nil
}

// detectInstallMethod inspects the resolved binary path to decide how the CLI
// was installed.
func detectInstallMethod(exePath string) string {
	p := strings.ToLower(filepath.ToSlash(exePath))
	switch {
	case strings.Contains(p, "node_modules"):
		return "npm"
	case strings.Contains(p, "/cellar/"), strings.Contains(p, "/homebrew/"), strings.Contains(p, "/linuxbrew/"):
		return "brew"
	default:
		return "direct"
	}
}

// runManagerUpdate delegates the update to an external package manager.
func runManagerUpdate(name string, args []string) error {
	bin, err := exec.LookPath(name)
	if err != nil {
		cmdline := name + " " + strings.Join(args, " ")
		output.Error(fmt.Sprintf("%s was not found on your PATH. Update manually:\n    %s", name, cmdline))
		return err
	}

	output.Info("Running: " + name + " " + strings.Join(args, " "))
	c := exec.Command(bin, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		output.Error("Update command failed: " + err.Error())
		return err
	}
	output.SuccessMsg("Update complete.")
	return nil
}
