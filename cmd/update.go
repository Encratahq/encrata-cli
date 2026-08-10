package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	Long: "Check for a newer release and update in place.\n\n" +
		"Detects how the CLI was installed (npm, Homebrew, or the direct install\n" +
		"script) and uses the matching update method automatically.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate()
	},
}

func runUpdate() error {
	output.Header("Update Encrata CLI")
	fmt.Printf("  Current version: v%s\n", version.Version)

	latest, err := latestRelease()
	if err != nil {
		output.Error("Could not check for updates: " + err.Error())
		return err
	}
	fmt.Printf("  Latest version:  v%s\n\n", latest)

	if version.Version != "dev" && normalizeVersion(latest) == normalizeVersion(version.Version) {
		output.SuccessMsg("You're already on the latest version.")
		return nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not locate the running binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}

	switch detectInstallMethod(exePath) {
	case "npm":
		return runManagerUpdate("npm", []string{"install", "-g", npmPackage + "@latest"})
	case "brew":
		return runManagerUpdate("brew", []string{"upgrade", "encrata"})
	default:
		return selfUpdate(exePath, latest)
	}
}

// latestRelease returns the latest release version (without a leading "v").
func latestRelease() (string, error) {
	url := "https://api.github.com/repos/" + updateRepo + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "encrata-cli")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("no release tag found")
	}
	return normalizeVersion(rel.TagName), nil
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
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
