package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var cliBinary string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp(filepath.Join("..", ".."), ".e2e-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)
	tmpDir, err = filepath.Abs(tmpDir)
	if err != nil {
		panic(err)
	}

	exe := "encrata-e2e"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	cliBinary = filepath.Join(tmpDir, exe)

	build := exec.Command("go", "build", "-buildvcs=false", "-o", cliBinary, ".")
	build.Dir = filepath.Join("..", "..")
	build.Env = append(os.Environ(), "GOCACHE="+filepath.Join(tmpDir, "gocache"))
	if out, err := build.CombinedOutput(); err != nil {
		panic("failed to build CLI for e2e tests: " + err.Error() + "\n" + string(out))
	}

	os.Exit(m.Run())
}

func TestVersionCommand(t *testing.T) {
	out, err := runCLI(t, nil, "version")
	if err != nil {
		t.Fatalf("version failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "encrata") {
		t.Fatalf("expected version output to mention encrata, got:\n%s", out)
	}
}

func TestFaceRejectsInvalidThreshold(t *testing.T) {
	env := map[string]string{
		"ENCRATA_API_KEY": "test-key",
	}

	out, err := runCLI(t, env, "face", "https://example.com/photo.jpg", "--threshold", "1.5")
	if err == nil {
		t.Fatalf("expected invalid threshold to fail, got success:\n%s", out)
	}
	if !strings.Contains(out, "threshold must be between 0 and 1") {
		t.Fatalf("expected threshold validation error, got:\n%s", out)
	}
}

func TestBulkLookupFromFileUsesSSEServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agent/bulk-lookup" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")

		// Default SSE events should be shown by the CLI.
		_, _ = w.Write([]byte("data: {\"data\":{\"email\":\"one@example.com\",\"person\":{\"first_name\":\"One\",\"last_name\":\"User\",\"company\":\"Acme\"}}}\n\n"))
		_, _ = w.Write([]byte("data: {\"data\":{\"email\":\"two@example.com\",\"person\":{\"first_name\":\"Two\",\"last_name\":\"User\",\"company\":\"Acme\"}}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	emailFile := filepath.Join(t.TempDir(), "emails.txt")
	if err := os.WriteFile(emailFile, []byte("one@example.com\ntwo@example.com\n"), 0o644); err != nil {
		t.Fatalf("failed to write email file: %v", err)
	}

	env := map[string]string{
		"ENCRATA_API_KEY":  "test-key",
		"ENCRATA_BASE_URL": server.URL,
	}

	out, err := runCLI(t, env, "bulk", "lookup", "--file", emailFile)
	if err != nil {
		t.Fatalf("bulk lookup failed: %v\n%s", err, out)
	}
	for _, want := range []string{"Bulk Lookup: 2 email(s)", "one@example.com", "two@example.com", "2 result(s)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func runCLI(t *testing.T, env map[string]string, args ...string) (string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cliBinary, args...)
	cmd.Env = append(os.Environ(),
		"ENCRATA_API_KEY=",
		"ENCRATA_BASE_URL=",
	)
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return string(out), ctx.Err()
	}
	return string(out), err
}
