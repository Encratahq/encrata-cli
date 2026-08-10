package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/Encratahq/cli/internal/output"
)

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

	if runtime.GOOS == "windows" {
		return replaceExecutableWindows(exePath, tmpPath)

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
	// this process runs; it will be removed on the next update.
	_ = os.Remove(oldPath)
	return nil
}

func replaceExecutableWindows(exePath, tmpPath string) error {
	scriptPath := tmpPath + ".bat"
	pid := os.Getpid()

	script := fmt.Sprintf(`@echo off
setlocal
:wait
tasklist /FI "PID eq %d" | find "%d" >nul
if not errorlevel 1 (
  timeout /T 1 /NOBREAK >nul
  goto wait
)
move /Y %s %s >nul
del /F /Q %s >nul 2>nul
del "%%~f0"
`, pid, pid, quoteBatchArg(tmpPath), quoteBatchArg(exePath), quoteBatchArg(exePath+".old"))

	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("could not create update script: %w", err)
	}

	cmd := exec.Command("cmd", "/C", "start", "/B", "", scriptPath)
	if err := cmd.Start(); err != nil {
		os.Remove(scriptPath)
		os.Remove(tmpPath)
		return fmt.Errorf("could not start update script: %w", err)
	}

	output.Info("Update will finish after this process exits.")
	return nil
}

func quoteBatchArg(s string) string {
	return strconv.Quote(s)
}
