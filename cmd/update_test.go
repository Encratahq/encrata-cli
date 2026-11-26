package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
		ok   bool
	}{
		{"0.14.3", "0.14.3", 0, true},
		{"v0.14.3", "0.14.3", 0, true},
		{"0.14.3", "0.15.0", -1, true},
		{"0.15.0", "0.14.3", 1, true},
		{"0.9.0", "0.10.0", -1, true}, // numeric, not lexical
		{"1.0.0", "0.99.99", 1, true},
		{"1.0.0-rc.1", "1.0.0", -1, true},
		{"1.0.0", "1.0.0-rc.1", 1, true},
		{"1.0.0-rc.1", "1.0.0-rc.2", -1, true},
		{"1.0.0+build.5", "1.0.0", 0, true},
		{"dev", "0.14.3", 0, false},
		{"0.14", "0.14.3", 0, false},
		{"0.14.x", "0.14.3", 0, false},
		{"1.0.0-", "1.0.0", 0, false},
	}
	for _, c := range cases {
		got, ok := compareVersions(c.a, c.b)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("compareVersions(%q, %q) = %d, %v; want %d, %v", c.a, c.b, got, ok, c.want, c.ok)
		}
	}
}

func TestChecksumFor(t *testing.T) {
	sum := strings.Repeat("ab", 32)
	file := []byte(sum + "  encrata_1.0.0_linux_amd64.tar.gz\n" +
		strings.Repeat("cd", 32) + " *encrata_1.0.0_windows_amd64.zip\n")
	if got, err := checksumFor(file, "encrata_1.0.0_linux_amd64.tar.gz"); err != nil || got != sum {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, err := checksumFor(file, "encrata_1.0.0_windows_amd64.zip"); err != nil || got != strings.Repeat("cd", 32) {
		t.Fatalf("binary-mode entry: got %q, %v", got, err)
	}
	if _, err := checksumFor(file, "encrata_1.0.0_darwin_arm64.tar.gz"); !errors.Is(err, errNoChecksum) {
		t.Fatalf("missing asset: err = %v", err)
	}
	if _, err := checksumFor([]byte("abc  x.zip\n"), "x.zip"); err == nil {
		t.Fatal("short checksum must be refused")
	}
}

// withReleaseServer points the release endpoints at h for one test.
func withReleaseServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	prevWeb, prevAPI := releaseWebBase, releaseAPIBase
	releaseWebBase, releaseAPIBase = srv.URL, srv.URL
	t.Cleanup(func() {
		releaseWebBase, releaseAPIBase = prevWeb, prevAPI
		srv.Close()
	})
	return srv
}

func TestLatestReleasePrefersRedirect(t *testing.T) {
	apiCalls := 0
	withReleaseServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + updateRepo + "/releases/latest":
			w.Header().Set("Location", "https://github.com/"+updateRepo+"/releases/tag/v0.15.0")
			w.WriteHeader(http.StatusFound)
		default:
			apiCalls++
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	got, err := latestRelease(context.Background())
	if err != nil || got != "0.15.0" {
		t.Fatalf("latestRelease = %q, %v", got, err)
	}
	if apiCalls != 0 {
		t.Fatalf("the rate-limited API was called %d times", apiCalls)
	}
}

func TestLatestReleaseFallsBackToAPI(t *testing.T) {
	withReleaseServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/"+updateRepo+"/releases/latest" {
			fmt.Fprint(w, `{"tag_name":"v0.16.1"}`)
			return
		}
		w.WriteHeader(http.StatusOK) // no redirect
	}))
	got, err := latestRelease(context.Background())
	if err != nil || got != "0.16.1" {
		t.Fatalf("latestRelease = %q, %v", got, err)
	}
}

func TestLatestReleaseRateLimitMessage(t *testing.T) {
	withReleaseServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/repos/") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	_, err := latestRelease(context.Background())
	if err == nil || !strings.Contains(err.Error(), "rate limit") {
		t.Fatalf("err = %v", err)
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("archive bytes")
	sum := sha256.Sum256(data)
	const asset = "encrata_1.0.0_test.zip"
	sums := hex.EncodeToString(sum[:]) + "  " + asset + "\n"
	missing := false
	withReleaseServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if missing {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprint(w, sums)
	}))
	ctx := context.Background()
	if err := verifyChecksum(ctx, "1.0.0", asset, data); err != nil {
		t.Fatalf("matching archive: %v", err)
	}
	if err := verifyChecksum(ctx, "1.0.0", asset, []byte("tampered")); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("tampered archive: err = %v", err)
	}
	if err := verifyChecksum(ctx, "1.0.0", "other.zip", data); !errors.Is(err, errNoChecksum) {
		t.Fatalf("unlisted asset: err = %v", err)
	}
	missing = true
	if err := verifyChecksum(ctx, "1.0.0", asset, data); err == nil {
		t.Fatal("a release without checksums.txt must fail closed")
	}
}

func TestExtractBinary(t *testing.T) {
	var zbuf bytes.Buffer
	zw := zip.NewWriter(&zbuf)
	f, _ := zw.Create("README.md")
	f.Write([]byte("readme"))
	f, _ = zw.Create("encrata.exe")
	f.Write([]byte("zip-binary"))
	zw.Close()
	if got, err := extractBinary(zbuf.Bytes(), "encrata.exe"); err != nil || string(got) != "zip-binary" {
		t.Fatalf("zip: %q, %v", got, err)
	}

	var tbuf bytes.Buffer
	gz := gzip.NewWriter(&tbuf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "encrata", Mode: 0o755, Size: 10})
	tw.Write([]byte("tar-binary"))
	tw.Close()
	gz.Close()
	if got, err := extractBinary(tbuf.Bytes(), "encrata"); err != nil || string(got) != "tar-binary" {
		t.Fatalf("tar.gz: %q, %v", got, err)
	}

	if _, err := extractBinary(zbuf.Bytes(), "missing"); err == nil {
		t.Fatal("expected a not-found error")
	}
}

func TestPreviousPath(t *testing.T) {
	if got := previousPath(filepath.Join("d", "encrata.exe")); got != filepath.Join("d", "encrata.previous.exe") {
		t.Errorf("windows: %s", got)
	}
	if got := previousPath(filepath.Join("d", "encrata")); got != filepath.Join("d", "encrata.previous") {
		t.Errorf("unix: %s", got)
	}
}

// stubVerify replaces the post-install check for one test.
func stubVerify(t *testing.T, fn func(path, want string) error) {
	t.Helper()
	prev := verifyBinary
	verifyBinary = fn
	t.Cleanup(func() { verifyBinary = prev })
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestInstallBinarySwapsAndKeepsPrevious(t *testing.T) {
	stubVerify(t, func(string, string) error { return nil })
	exe := filepath.Join(t.TempDir(), "encrata.exe")
	os.WriteFile(exe, []byte("v1"), 0o755)

	if err := installBinary(exe, []byte("v2"), "2.0.0"); err != nil {
		t.Fatal(err)
	}
	if readFile(t, exe) != "v2" || readFile(t, previousPath(exe)) != "v1" {
		t.Fatalf("after update: exe=%q previous=%q", readFile(t, exe), readFile(t, previousPath(exe)))
	}

	// A second update replaces the backup with the version it replaces.
	if err := installBinary(exe, []byte("v3"), "3.0.0"); err != nil {
		t.Fatal(err)
	}
	if readFile(t, exe) != "v3" || readFile(t, previousPath(exe)) != "v2" {
		t.Fatalf("after second update: exe=%q previous=%q", readFile(t, exe), readFile(t, previousPath(exe)))
	}
	assertNoTempFiles(t, filepath.Dir(exe))
}

func TestInstallBinaryRestoresWhenNewBinaryFails(t *testing.T) {
	stubVerify(t, func(string, string) error { return errors.New("exec format error") })
	exe := filepath.Join(t.TempDir(), "encrata")
	os.WriteFile(exe, []byte("good"), 0o755)

	err := installBinary(exe, []byte("broken"), "2.0.0")
	if err == nil || !strings.Contains(err.Error(), "previous version was kept") {
		t.Fatalf("err = %v", err)
	}
	if readFile(t, exe) != "good" {
		t.Fatalf("the working binary was not restored: %q", readFile(t, exe))
	}
	if _, err := os.Stat(previousPath(exe)); !os.IsNotExist(err) {
		t.Fatalf("no backup should remain after a restore: %v", err)
	}
	assertNoTempFiles(t, filepath.Dir(exe))
}

func TestInstallBinaryReadOnlyDirectoryLeavesBinaryAlone(t *testing.T) {
	stubVerify(t, func(string, string) error { return nil })
	exe := filepath.Join(t.TempDir(), "missing-dir", "encrata")
	if err := installBinary(exe, []byte("v2"), "2.0.0"); err == nil {
		t.Fatal("expected an error for an unwritable install directory")
	}
}

func TestPermissionHint(t *testing.T) {
	err := permissionHint("/usr/local/bin", fmt.Errorf("open: %w", os.ErrPermission))
	want := "sudo"
	if runtime.GOOS == "windows" {
		want = "Administrator"
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("hint = %v", err)
	}
	if err := permissionHint("/x", errors.New("disk full")); strings.Contains(err.Error(), want) {
		t.Fatalf("non-permission errors must not suggest %s: %v", want, err)
	}
}

func TestRollbackBinarySwapsBothWays(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "encrata.exe")
	os.WriteFile(exe, []byte("new"), 0o755)

	if err := rollbackBinary(exe); err == nil || !strings.Contains(err.Error(), "nothing to roll back") {
		t.Fatalf("without a backup: err = %v", err)
	}

	os.WriteFile(previousPath(exe), []byte("old"), 0o755)
	if err := rollbackBinary(exe); err != nil {
		t.Fatal(err)
	}
	if readFile(t, exe) != "old" || readFile(t, previousPath(exe)) != "new" {
		t.Fatalf("after rollback: exe=%q previous=%q", readFile(t, exe), readFile(t, previousPath(exe)))
	}
	if err := rollbackBinary(exe); err != nil {
		t.Fatal(err)
	}
	if readFile(t, exe) != "new" || readFile(t, previousPath(exe)) != "old" {
		t.Fatal("a second rollback must undo the first")
	}
	assertNoTempFiles(t, filepath.Dir(exe))
}

func TestUpdateLeftovers(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "encrata.exe")
	os.WriteFile(exe, []byte("x"), 0o755)
	old := time.Now().Add(-2 * leftoverMinAge)

	stale := filepath.Join(dir, updateStalePrefix+"123")
	tmpOld := filepath.Join(dir, updateTempPrefix+"abc")
	tmpFresh := filepath.Join(dir, updateTempPrefix+"inflight")
	legacy := exe + ".old"
	for _, p := range []string{stale, tmpOld, tmpFresh, legacy} {
		os.WriteFile(p, []byte("x"), 0o600)
	}
	os.Chtimes(stale, old, old)
	os.Chtimes(tmpOld, old, old)

	got := map[string]bool{}
	for _, p := range updateLeftovers(exe) {
		got[p] = true
	}
	if !got[stale] || !got[tmpOld] || !got[legacy] {
		t.Fatalf("missing leftovers: %v", got)
	}
	if got[tmpFresh] {
		t.Fatal("a fresh temp file may belong to an update in progress and must be left alone")
	}
	if got[exe] || got[previousPath(exe)] {
		t.Fatal("the binary and its backup are never leftovers")
	}
}

func TestRetireMissingFile(t *testing.T) {
	if err := retire(filepath.Join(t.TempDir(), "nope")); err != nil {
		t.Fatal(err)
	}
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	for _, pattern := range []string{updateTempPrefix + "*", updateStalePrefix + "*"} {
		if m, _ := filepath.Glob(filepath.Join(dir, pattern)); len(m) > 0 {
			t.Fatalf("left behind: %v", m)
		}
	}
}

// ── update notice ───────────────────────────────────────────────────────────

func TestUpdateNoticeEnabled(t *testing.T) {
	env := func(kv map[string]string) func(string) string {
		return func(k string) string { return kv[k] }
	}
	none := env(nil)
	cases := []struct {
		name   string
		args   []string
		getenv func(string) string
		tty    bool
		ver    string
		want   bool
	}{
		{"interactive lookup", []string{"email", "validity", "a@b.co"}, none, true, "0.14.3", true},
		{"not a terminal", []string{"whoami"}, none, false, "0.14.3", false},
		{"CI", []string{"whoami"}, env(map[string]string{"CI": "true"}), true, "0.14.3", false},
		{"opt-out", []string{"whoami"}, env(map[string]string{envNoUpdateNotifier: "1"}), true, "0.14.3", false},
		{"dev build", []string{"whoami"}, none, true, "dev", false},
		{"update itself", []string{"update"}, none, true, "0.14.3", false},
		{"version", []string{"version"}, none, true, "0.14.3", false},
		{"doctor", []string{"doctor"}, none, true, "0.14.3", false},
		{"completion", []string{"completion", "bash"}, none, true, "0.14.3", false},
	}
	for _, c := range cases {
		if got := updateNoticeEnabled(c.args, c.getenv, c.tty, c.ver); got != c.want {
			t.Errorf("%s: enabled = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestUpdateNoticeLine(t *testing.T) {
	if line, ok := updateNoticeLine("0.14.3", "0.15.0"); !ok || !strings.Contains(line, "v0.14.3 → v0.15.0") || !strings.Contains(line, "encrata update") {
		t.Fatalf("newer: %q %v", line, ok)
	}
	for _, latest := range []string{"0.14.3", "0.14.2", "", "garbage"} {
		if _, ok := updateNoticeLine("0.14.3", latest); ok {
			t.Errorf("latest %q must not show a notice", latest)
		}
	}
}

func TestUpdateCacheRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if _, ok := loadUpdateCache(); ok {
		t.Fatal("empty home must have no cache")
	}
	saveUpdateCache("0.15.0")
	c, ok := loadUpdateCache()
	if !ok || c.Latest != "0.15.0" || time.Since(c.CheckedAt) > time.Minute {
		t.Fatalf("cache = %+v, %v", c, ok)
	}
}

// ── doctor ──────────────────────────────────────────────────────────────────

func TestFindOnPathAndPathCheck(t *testing.T) {
	name := "encrata"
	if runtime.GOOS == "windows" {
		name = "encrata.exe"
	}
	a, b, empty := t.TempDir(), t.TempDir(), t.TempDir()
	exeA, exeB := filepath.Join(a, name), filepath.Join(b, name)
	os.WriteFile(exeA, []byte("x"), 0o755)
	os.WriteFile(exeB, []byte("x"), 0o755)
	sep := string(os.PathListSeparator)

	found := findOnPath(strings.Join([]string{empty, a, a, b}, sep))
	if len(found) != 2 || found[0] != exeA || found[1] != exeB {
		t.Fatalf("found = %v", found)
	}

	if c := pathCheck(exeA, "direct", found); c.Status != checkWarn || !strings.Contains(c.Detail, "2 copies") {
		t.Fatalf("duplicates: %+v", c)
	}
	if c := pathCheck(exeA, "direct", found[:1]); c.Status != checkOK {
		t.Fatalf("single matching install: %+v", c)
	}
	if c := pathCheck(exeB, "direct", found[:1]); c.Status != checkWarn || !strings.Contains(c.Detail, "not this binary") {
		t.Fatalf("PATH runs another binary: %+v", c)
	}
	if c := pathCheck(exeB, "npm", found[:1]); c.Status != checkOK {
		t.Fatalf("npm's PATH entry is a launcher and must not be compared: %+v", c)
	}
	if c := pathCheck(exeA, "direct", nil); c.Status != checkWarn {
		t.Fatalf("not on PATH: %+v", c)
	}
}

func TestMaskKey(t *testing.T) {
	if got := maskKey("enc_live_1234567890abcd"); got != "enc_live…abcd" {
		t.Errorf("maskKey = %q", got)
	}
	if got := maskKey("short"); strings.Contains(got, "short") {
		t.Errorf("a short key must be fully masked: %q", got)
	}
}
