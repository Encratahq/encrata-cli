package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Encratahq/cli/internal/output"
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
	fmt.Printf("  Current version: v%s\n", version)

	latest, err := latestRelease()
	if err != nil {
		output.Error("Could not check for updates: " + err.Error())
		return err
	}
	fmt.Printf("  Latest version:  v%s\n\n", latest)

	if version != "dev" && normalizeVersion(latest) == normalizeVersion(version) {
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

// selfUpdate downloads the release archive and replaces the running binary.
// Used when the CLI was installed via the direct install script.
func selfUpdate(exePath, version string) error {
	platform := runtime.GOOS // darwin, linux, windows
	arch := runtime.GOARCH   // amd64, arm64

	ext := ".tar.gz"
	binName := "encrata"
	if platform == "windows" {
		ext = ".zip"
		binName = "encrata.exe"
	}

	asset := fmt.Sprintf("encrata_%s_%s_%s%s", version, platform, arch, ext)
	url := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", updateRepo, version, asset)

	output.Info("Downloading " + asset + "...")
	archive, err := download(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	binData, err := extractBinary(archive, ext, binName)
	if err != nil {
		return fmt.Errorf("could not extract binary: %w", err)
	}

	if err := replaceExecutable(exePath, binData); err != nil {
		return err
	}

	output.SuccessMsg("Updated to v" + version + ".")
	return nil
}

func download(url string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "encrata-cli")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// extractBinary pulls the encrata binary out of a .zip or .tar.gz archive.
func extractBinary(archive []byte, ext, binName string) ([]byte, error) {
	if ext == ".zip" {
		zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if filepath.Base(f.Name) == binName {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
		return nil, fmt.Errorf("%s not found in archive", binName)
	}

	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) == binName {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("%s not found in archive", binName)
}

func replaceExecutable(exePath string, data []byte) error {
	dir := filepath.Dir(exePath)

	tmp, err := os.CreateTemp(dir, ".encrata-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to install directory %q: %w", dir, err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		os.Remove(tmpPath)
		return err
	}

	oldPath := exePath + ".old"
	_ = os.Remove(oldPath) // clear any leftover from a previous update

	if err := os.Rename(exePath, oldPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("could not move current binary aside: %w", err)
	}
	if err := os.Rename(tmpPath, exePath); err != nil {
		_ = os.Rename(oldPath, exePath) // best-effort rollback
		os.Remove(tmpPath)
		return fmt.Errorf("could not install new binary: %w", err)
	}

	// Best-effort cleanup. On Windows the old binary may still be locked while

	_ = os.Remove(oldPath)
	return nil
}
