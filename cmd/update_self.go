package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Encratahq/cli/internal/output"
)

// How a direct (non-package-manager) install is replaced
//
// The new binary is written next to the current one, then swapped in with two
// renames: current → "<name>.previous", new → "<name>". A rename works on every
// OS even while the binary is running (Windows locks a running .exe against
// deletion and overwrite, but not against rename), so the update is complete
// when the command returns. The new binary is then run once (`version`); if it
// does not start, the swap is undone. "<name>.previous" is kept for
// `encrata update --rollback`. Anything a crash or a locked file leaves behind
// is renamed to a ".encrata-stale-*" name and swept on a later run.

// Leftover-file name patterns in the install directory.
const (
	updateTempPrefix  = ".encrata-update-"
	updateStalePrefix = ".encrata-stale-"
	// leftoverMinAge keeps the sweep away from an update another process is
	// running right now.
	leftoverMinAge = time.Hour
)

// verifyBinary runs a freshly installed binary and checks it reports want. A
// variable so tests can stub it when the "binary" is not executable.
var verifyBinary = runVersionCheck

// selfUpdate downloads the release archive for this platform, verifies it, and
// swaps it in for the running binary.
func selfUpdate(ctx context.Context, exePath, current, latest string) error {
	asset, binName := releaseAsset(latest)

	output.Info("Downloading " + asset + "...")
	archive, err := download(ctx, releaseAssetURL(latest, asset))
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	if err := verifyChecksum(ctx, latest, asset, archive); err != nil {
		return err
	}
	output.Info("Checksum verified.")

	binData, err := extractBinary(archive, binName)
	if err != nil {
		return fmt.Errorf("could not extract binary: %w", err)
	}
	if err := installBinary(exePath, binData, latest); err != nil {
		return err
	}

	output.SuccessMsg(fmt.Sprintf("Updated v%s → v%s.", current, latest))
	output.Dim.Println("  The previous version is kept; undo with: encrata update --rollback")
	return nil
}

// releaseAsset returns the archive name for this platform and the binary name
// inside it (GoReleaser's encrata_<version>_<os>_<arch>.{tar.gz,zip}).
func releaseAsset(version string) (asset, binName string) {
	ext, binName := ".tar.gz", "encrata"
	if runtime.GOOS == "windows" {
		ext, binName = ".zip", "encrata.exe"
	}
	return fmt.Sprintf("encrata_%s_%s_%s%s", version, runtime.GOOS, runtime.GOARCH, ext), binName
}

// extractBinary pulls binName out of a .zip or .tar.gz archive.
func extractBinary(archive []byte, binName string) ([]byte, error) {
	if zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive))); err == nil {
		for _, f := range zr.File {
			if filepath.Base(f.Name) == binName {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(io.LimitReader(rc, maxReleaseDownload))
			}
		}
		return nil, fmt.Errorf("%s not found in archive", binName)
	}

	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("archive is neither zip nor tar.gz: %w", err)
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
			return io.ReadAll(io.LimitReader(tr, maxReleaseDownload))
		}
	}
	return nil, fmt.Errorf("%s not found in archive", binName)
}

// previousPath is where the replaced binary is kept: encrata.previous, or
// encrata.previous.exe on Windows (so it stays runnable by hand).
func previousPath(exePath string) string {
	ext := filepath.Ext(exePath)
	if !strings.EqualFold(ext, ".exe") {
		ext = ""
	}
	return strings.TrimSuffix(exePath, ext) + ".previous" + ext
}

// installBinary writes data next to exePath and swaps it in, keeping the old
// binary at previousPath. On any failure the original binary is left (or put
// back) in place.
func installBinary(exePath string, data []byte, want string) error {
	dir := filepath.Dir(exePath)
	tmp, err := os.CreateTemp(dir, updateTempPrefix+"*")
	if err != nil {
		return permissionHint(dir, err)
	}
	tmpPath := tmp.Name()
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Chmod(tmpPath, 0o755)
	}
	if werr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("could not write the new binary: %w", werr)
	}

	prev := previousPath(exePath)
	if err := retire(prev); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("could not clear the old backup %s: %w", prev, err)
	}
	if err := os.Rename(exePath, prev); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("could not move the current binary aside: %w", err)
	}
	if err := os.Rename(tmpPath, exePath); err != nil {
		os.Remove(tmpPath)
		if rerr := os.Rename(prev, exePath); rerr != nil {
			return fmt.Errorf("could not install the new binary (%v) and could not restore the old one (%v); restore it by renaming %s to %s", err, rerr, prev, exePath)
		}
		return fmt.Errorf("could not install the new binary: %w", err)
	}

	if err := verifyBinary(exePath, want); err != nil {
		if rerr := restorePrevious(exePath, prev); rerr != nil {
			return fmt.Errorf("the new binary failed to start (%v) and the old one could not be restored (%v); restore it by renaming %s to %s", err, rerr, prev, exePath)
		}
		return fmt.Errorf("the new binary failed to start, so the previous version was kept: %w", err)
	}
	return nil
}

// restorePrevious undoes a swap: the (not running) new binary is retired and the
// previous one renamed back.
func restorePrevious(exePath, prev string) error {
	if err := retire(exePath); err != nil {
		return err
	}
	return os.Rename(prev, exePath)
}

// rollbackBinary swaps exePath and its previous version. Every step is a
// rename, so it works while exePath is running, on Windows too.
func rollbackBinary(exePath string) error {
	prev := previousPath(exePath)
	if _, err := os.Stat(prev); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("no previous version is kept next to %s, so there is nothing to roll back to", exePath)
		}
		return err
	}
	stage := filepath.Join(filepath.Dir(exePath), fmt.Sprintf("%srollback-%d", updateStalePrefix, time.Now().UnixNano()))
	if err := os.Rename(exePath, stage); err != nil {
		return permissionHint(filepath.Dir(exePath), err)
	}
	if err := os.Rename(prev, exePath); err != nil {
		_ = os.Rename(stage, exePath)
		return fmt.Errorf("could not restore the previous version: %w", err)
	}
	// The rolled-back-from version becomes the new "previous", so a second
	// rollback returns to it.
	if err := os.Rename(stage, prev); err != nil {
		output.Dim.Println("  Note: the version you rolled back from could not be kept: " + err.Error())
	}
	return nil
}

// retire removes path. If it cannot be deleted (on Windows, because a process is
// still running from it) it is renamed to a stale name swept on a later run.
func retire(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	stale := filepath.Join(filepath.Dir(path), fmt.Sprintf("%s%d", updateStalePrefix, time.Now().UnixNano()))
	if rerr := os.Rename(path, stale); rerr != nil {
		return err
	}
	return nil
}

// runVersionCheck runs `<path> version` and checks it reports want.
func runVersionCheck(path, want string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, path, "version")
	c.Env = append(os.Environ(), envNoUpdateNotifier+"=1")
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("`%s version`: %v", filepath.Base(path), err)
	}
	if !strings.Contains(string(out), "v"+want) {
		return fmt.Errorf("it reported %q, expected v%s", strings.TrimSpace(stripANSI(string(out))), want)
	}
	return nil
}

// permissionHint explains how to update when the install directory is not
// writable by this user.
func permissionHint(dir string, err error) error {
	if !errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("cannot write to the install directory %s: %w", dir, err)
	}
	hint := "re-run with sudo (sudo encrata update), or reinstall into a directory you own"
	if runtime.GOOS == "windows" {
		hint = "run the terminal as Administrator, or reinstall per-user with install.ps1 (it installs to %LOCALAPPDATA%\\Programs\\Encrata)"
	}
	return fmt.Errorf("no permission to write to %s: %s", dir, hint)
}

// sweepUpdateLeftovers deletes what earlier updates could not: stale renamed
// binaries, interrupted temp files, and the ".old" file older CLI versions
// left. Best-effort and cheap: two globs in the install directory.
func sweepUpdateLeftovers() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}
	for _, p := range updateLeftovers(exePath) {
		_ = os.Remove(p) // a file still in use stays until a later run
	}
}

// updateLeftovers lists the sweepable files next to exePath.
func updateLeftovers(exePath string) []string {
	dir := filepath.Dir(exePath)
	var out []string
	for _, pattern := range []string{updateStalePrefix + "*", updateTempPrefix + "*"} {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		for _, m := range matches {
			if info, err := os.Stat(m); err == nil && !info.IsDir() && time.Since(info.ModTime()) >= leftoverMinAge {
				out = append(out, m)
			}
		}
	}
	// Written by CLI versions up to 0.14.x.
	if _, err := os.Stat(exePath + ".old"); err == nil {
		out = append(out, exePath+".old")
	}
	return out
}

// stripANSI removes color escapes for error messages.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < '@' || s[j] > '~') {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
