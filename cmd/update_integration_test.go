package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// buildCLI compiles the CLI with version v into dir.
func buildCLI(t *testing.T, dir, name, v string) string {
	t.Helper()
	out := filepath.Join(dir, name)
	build := exec.Command("go", "build", "-buildvcs=false",
		"-ldflags", "-X github.com/Encratahq/cli/internal/version.Version="+v,
		"-o", out, ".")
	build.Dir = ".."
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build v%s: %v\n%s", v, err, b)
	}
	return out
}

// releaseArchive packages bin the way GoReleaser does for this platform.
func releaseArchive(t *testing.T, bin []byte, binName string) []byte {
	t.Helper()
	var buf bytes.Buffer
	if runtime.GOOS == "windows" {
		zw := zip.NewWriter(&buf)
		f, _ := zw.Create(binName)
		f.Write(bin)
		zw.Close()
		return buf.Bytes()
	}
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: binName, Mode: 0o755, Size: int64(len(bin))})
	tw.Write(bin)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

// TestSelfUpdateEndToEnd updates a real, running v0.0.1 binary to v9.9.9 from a
// local release server, then rolls back, all while the old process is still
// running (the case Windows used to need a deferred .bat for).
func TestSelfUpdateEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI twice")
	}
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	build := t.TempDir()
	install := t.TempDir()
	newBin, err := os.ReadFile(buildCLI(t, build, "new"+ext, "9.9.9"))
	if err != nil {
		t.Fatal(err)
	}
	exe := buildCLI(t, install, "encrata"+ext, "0.0.1")

	asset, binName := releaseAsset("9.9.9")
	archive := releaseArchive(t, newBin, binName)
	sum := sha256.Sum256(archive)
	withReleaseServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + updateRepo + "/releases/latest":
			w.Header().Set("Location", "/"+updateRepo+"/releases/tag/v9.9.9")
			w.WriteHeader(http.StatusFound)
		case "/" + updateRepo + "/releases/download/v9.9.9/" + asset:
			w.Write(archive)
		case "/" + updateRepo + "/releases/download/v9.9.9/checksums.txt":
			fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), asset)
		default:
			http.NotFound(w, r)
		}
	}))

	// Keep the old binary running: `login` waits on STDIN for a key.
	running := exec.Command(exe, "login")
	running.Env = append(os.Environ(), envNoUpdateNotifier+"=1", "HOME="+install, "USERPROFILE="+install)
	stdin, err := running.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := running.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		stdin.Close()
		done := make(chan struct{})
		go func() { running.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			running.Process.Kill()
		}
	}()
	time.Sleep(300 * time.Millisecond) // let it load and lock its image

	ctx := context.Background()
	latest, err := latestRelease(ctx)
	if err != nil || latest != "9.9.9" {
		t.Fatalf("latestRelease = %q, %v", latest, err)
	}
	if err := selfUpdate(ctx, exe, "0.0.1", latest); err != nil {
		t.Fatalf("selfUpdate while the old binary runs: %v", err)
	}
	if got := runVersion(t, exe); !strings.Contains(got, "v9.9.9") {
		t.Fatalf("after update, version = %q", got)
	}
	if got := runVersion(t, previousPath(exe)); !strings.Contains(got, "v0.0.1") {
		t.Fatalf("previous version = %q", got)
	}

	if err := rollbackBinary(exe); err != nil {
		t.Fatalf("rollback while a binary runs: %v", err)
	}
	if got := runVersion(t, exe); !strings.Contains(got, "v0.0.1") {
		t.Fatalf("after rollback, version = %q", got)
	}
	if got := runVersion(t, previousPath(exe)); !strings.Contains(got, "v9.9.9") {
		t.Fatalf("after rollback, previous = %q", got)
	}

	// A corrupt download is refused and leaves the install untouched.
	archive = append([]byte(nil), archive...)
	archive[len(archive)/2] ^= 0xff
	if err := selfUpdate(ctx, exe, "0.0.1", latest); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("corrupt archive: err = %v", err)
	}
	if got := runVersion(t, exe); !strings.Contains(got, "v0.0.1") {
		t.Fatalf("after a refused update, version = %q", got)
	}
}

func runVersion(t *testing.T, path string) string {
	t.Helper()
	c := exec.Command(path, "version")
	c.Env = append(os.Environ(), envNoUpdateNotifier+"=1")
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("%s version: %v\n%s", path, err, out)
	}
	return stripANSI(string(out))
}

