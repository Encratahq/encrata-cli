package e2e

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
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

func TestEmailValidityRejectsInvalidAddress(t *testing.T) {
	env := map[string]string{
		"ENCRATA_API_KEY": "test-key",
	}

	out, err := runCLI(t, env, "email", "validity", "not-an-email")
	if err == nil {
		t.Fatalf("expected invalid email to fail, got success:\n%s", out)
	}
	if !strings.Contains(out, "invalid email address") {
		t.Fatalf("expected email validation error, got:\n%s", out)
	}
}

func TestEmailBulkFromFileUsesSSEServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/bulk-validity-search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")

		_, _ = w.Write([]byte("event: start\ndata: {\"total\":2}\n\n"))
		_, _ = w.Write([]byte("data: {\"email\":\"one@example.com\",\"validity\":\"valid\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"email\":\"two@example.com\",\"validity\":\"invalid\"}\n\n"))
		_, _ = w.Write([]byte("event: done\ndata: {\"processed\":2}\n\n"))
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

	out, err := runCLI(t, env, "email", "bulk", emailFile, "--stream")
	if err != nil {
		t.Fatalf("email bulk failed: %v\n%s", err, out)
	}
	for _, want := range []string{"Bulk Validity: 2 emails", "Total", "Valid"} {
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

// exitCode extracts the process exit code from a runCLI error (0 when nil).
func exitCode(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	t.Fatalf("expected an exit error, got: %v", err)
	return -1
}

func TestPasswordSingleBreachedExitsOne(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/password/breaches" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if _, ok := body["password"]; ok {
			t.Fatalf("plaintext password must never be sent")
		}
		// SHA-1 of "password", upper-case hex.
		if body["sha1"] != "5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8" {
			t.Fatalf("expected upper-hex sha1, got %v", body["sha1"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"prefix":"5BAA6","found":true,"count":12345,"message":"unsafe","credits":1}`))
	}))
	defer server.Close()

	env := map[string]string{
		"ENCRATA_API_KEY":  "test-key",
		"ENCRATA_BASE_URL": server.URL,
	}

	out, err := runCLI(t, env, "password", "password")
	if code := exitCode(t, err); code != 1 {
		t.Fatalf("expected exit code 1 on breach, got %d\n%s", code, out)
	}
	for _, want := range []string{"BREACHED", "12,345", "5BAA6"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPasswordSingleSafeExitsZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"prefix":"ABCDE","found":false,"count":0,"message":"safe","credits":1}`))
	}))
	defer server.Close()

	env := map[string]string{
		"ENCRATA_API_KEY":  "test-key",
		"ENCRATA_BASE_URL": server.URL,
	}

	out, err := runCLI(t, env, "password", "s3cret!")
	if code := exitCode(t, err); code != 0 {
		t.Fatalf("expected exit code 0 when not breached, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Not found in any known breach") {
		t.Fatalf("expected safe verdict, got:\n%s", out)
	}
}

func TestPasswordJSONModePassesThrough(t *testing.T) {
	response := `{"prefix":"5BAA6","found":true,"count":42,"message":"unsafe","credits":1}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	env := map[string]string{
		"ENCRATA_API_KEY":  "test-key",
		"ENCRATA_BASE_URL": server.URL,
	}

	out, err := runCLI(t, env, "password", "password", "--json")
	if code := exitCode(t, err); code != 1 {
		t.Fatalf("expected exit code 1 on breach, got %d\n%s", code, out)
	}
	// JSON mode should emit the raw fields, not the table verdict line.
	for _, want := range []string{`"prefix": "5BAA6"`, `"found": true`, `"count": 42`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected JSON output to contain %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "BREACHED") {
		t.Fatalf("JSON mode should not render the table verdict, got:\n%s", out)
	}
}

func TestPasswordBulkFromFileDedupes(t *testing.T) {
	var gotHashes []interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/password/breaches/bulk" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if _, ok := body["passwords"]; ok {
			t.Fatalf("plaintext passwords must never be sent")
		}
		gotHashes, _ = body["sha1s"].([]interface{})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":2,"breached":1,"results":[{"prefix":"ABCDE","found":true,"count":42},{"prefix":"F0A21","found":false,"count":0}],"credits":2}`))
	}))
	defer server.Close()

	// Three lines, one duplicate → two unique hashes expected.
	pwFile := filepath.Join(t.TempDir(), "passwords.txt")
	if err := os.WriteFile(pwFile, []byte("alpha\nbeta\nalpha\n"), 0o644); err != nil {
		t.Fatalf("failed to write password file: %v", err)
	}

	env := map[string]string{
		"ENCRATA_API_KEY":  "test-key",
		"ENCRATA_BASE_URL": server.URL,
	}

	out, err := runCLI(t, env, "password", "--file", pwFile)
	if code := exitCode(t, err); code != 1 {
		t.Fatalf("expected exit code 1 (breach in batch), got %d\n%s", code, out)
	}
	if len(gotHashes) != 2 {
		t.Fatalf("expected 2 de-duplicated hashes, got %d: %v", len(gotHashes), gotHashes)
	}
	for _, want := range []string{"Prefix", "of 2 breached", "2 credits"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

// enrichedResult is a fully-populated bulk row used to exercise the flattened
// export columns (nested smtp/domain_trust/footprint/etc.).
const enrichedResult = `{"email":"rich@example.com","validity":"valid","reason":"deliverable","message":"ok",` +
	`"confidence":0.98,"disposable":false,"role":false,"free_provider":true,"provider":"google",` +
	`"canonical":"rich@example.com","domain":"example.com","mx":["mx1.example.com","mx2.example.com"],` +
	`"smtp":{"mx_host":"mx1.example.com","catch_all":false,"greylisted":false},` +
	`"domain_trust":{"grade":"A","spf":true,"dmarc":true,"dmarc_policy":"reject","dkim":true,"mta_sts":"enforce","tls_rpt":true,"bimi":false,"dnssec":true},` +
	`"person_signal":{"count":3,"sources":["github","linkedin"]},` +
	`"domain_info":{"registrar":"MarkMonitor","created_at":"2000-01-01","age_days":9000},` +
	`"footprint":{"breaches":{"count":2},"gravatar_url":"http://gravatar/x","registered_services":["a","b","c"],"google":true},` +
	`"checked_at":"2026-07-27T10:00:00Z"}`

const bareResult = `{"email":"bare@example.com","validity":"invalid","reason":"hard_reject"}`

// bulkSSEServer returns a server that streams the two fixture rows over SSE.
func bulkSSEServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/bulk-validity-search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: start\ndata: {\"total\":2}\n\n")
		_, _ = io.WriteString(w, "data: "+enrichedResult+"\n\n")
		_, _ = io.WriteString(w, "data: "+bareResult+"\n\n")
		_, _ = io.WriteString(w, "event: done\ndata: {\"processed\":2}\n\n")
	}))
}

func writeEmailsFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "emails.txt")
	if err := os.WriteFile(path, []byte("rich@example.com\nbare@example.com\n"), 0o644); err != nil {
		t.Fatalf("failed to write emails file: %v", err)
	}
	return path
}

func colIndex(header []string, name string) int {
	for i, h := range header {
		if h == name {
			return i
		}
	}
	return -1
}

func TestBulkExportCSVFlattensColumns(t *testing.T) {
	server := bulkSSEServer(t)
	defer server.Close()

	emails := writeEmailsFile(t)
	out := filepath.Join(t.TempDir(), "results.csv")
	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}

	if o, err := runCLI(t, env, "email", "bulk", emails, "--stream", "--out", out); err != nil {
		t.Fatalf("bulk export failed: %v\n%s", err, o)
	}

	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("failed to open csv: %v", err)
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse csv: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected header + 2 rows, got %d", len(records))
	}
	header := records[0]
	if header[0] != "email" || header[1] != "status" || header[2] != "reason" {
		t.Fatalf("unexpected leading columns: %v", header[:3])
	}
	for _, want := range []string{"trust_grade", "mx", "gravatar", "checked_at"} {
		if colIndex(header, want) < 0 {
			t.Fatalf("expected column %q in header: %v", want, header)
		}
	}

	rich := records[1]
	checks := map[string]string{
		"mx":                    "mx1.example.com | mx2.example.com",
		"disposable":            "no",
		"free_provider":         "yes",
		"trust_grade":           "A",
		"breaches_count":        "2",
		"registered_services":   "3",
		"gravatar":              "yes",
		"google_account":        "yes",
		"person_signal_sources": "github | linkedin",
	}
	for col, want := range checks {
		idx := colIndex(header, col)
		if idx < 0 {
			t.Fatalf("missing column %q", col)
		}
		if rich[idx] != want {
			t.Fatalf("column %q = %q, want %q", col, rich[idx], want)
		}
	}
}

func TestBulkExportColumnsSubset(t *testing.T) {
	server := bulkSSEServer(t)
	defer server.Close()

	emails := writeEmailsFile(t)
	out := filepath.Join(t.TempDir(), "subset.csv")
	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}

	if o, err := runCLI(t, env, "email", "bulk", emails, "--stream", "--out", out, "--columns", "trust_grade,domain"); err != nil {
		t.Fatalf("bulk export failed: %v\n%s", err, o)
	}

	f, _ := os.Open(out)
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse csv: %v", err)
	}
	// email/status/reason always included, then requested columns in canonical order.
	want := []string{"email", "status", "reason", "domain", "trust_grade"}
	if strings.Join(records[0], ",") != strings.Join(want, ",") {
		t.Fatalf("header = %v, want %v", records[0], want)
	}
}

func TestBulkExportFoundOnly(t *testing.T) {
	server := bulkSSEServer(t)
	defer server.Close()

	emails := writeEmailsFile(t)
	out := filepath.Join(t.TempDir(), "found.csv")
	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}

	if o, err := runCLI(t, env, "email", "bulk", emails, "--stream", "--out", out, "--found-only"); err != nil {
		t.Fatalf("bulk export failed: %v\n%s", err, o)
	}

	f, _ := os.Open(out)
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse csv: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected header + 1 enriched row, got %d rows", len(records))
	}
	if records[1][0] != "rich@example.com" {
		t.Fatalf("expected only the enriched row, got %v", records[1][0])
	}
}

func TestBulkExportValidOnly(t *testing.T) {
	server := bulkSSEServer(t)
	defer server.Close()

	emails := writeEmailsFile(t)
	out := filepath.Join(t.TempDir(), "valid-only.csv")
	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}

	if o, err := runCLI(t, env, "email", "bulk", emails, "--stream", "--out", out, "--valid-only"); err != nil {
		t.Fatalf("bulk export failed: %v\n%s", err, o)
	}

	f, _ := os.Open(out)
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse csv: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected header + 1 valid row, got %d rows", len(records))
	}
	if records[1][0] != "rich@example.com" {
		t.Fatalf("expected only the valid row, got %v", records[1][0])
	}
}

func TestBulkExportJSONRaw(t *testing.T) {
	server := bulkSSEServer(t)
	defer server.Close()

	emails := writeEmailsFile(t)
	out := filepath.Join(t.TempDir(), "results.json")
	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}

	if o, err := runCLI(t, env, "email", "bulk", emails, "--stream", "--out", out, "--format", "json"); err != nil {
		t.Fatalf("bulk export failed: %v\n%s", err, o)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("failed to read json: %v", err)
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 raw rows, got %d", len(rows))
	}
	// Raw JSON keeps the nested structure (unflattened).
	if _, ok := rows[0]["domain_trust"].(map[string]interface{}); !ok {
		t.Fatalf("expected nested domain_trust object in raw JSON, got: %v", rows[0]["domain_trust"])
	}
}

func TestBulkExportXLSX(t *testing.T) {
	server := bulkSSEServer(t)
	defer server.Close()

	emails := writeEmailsFile(t)
	out := filepath.Join(t.TempDir(), "results.xlsx")
	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}

	// Format is inferred from the .xlsx extension.
	if o, err := runCLI(t, env, "email", "bulk", emails, "--stream", "--out", out); err != nil {
		t.Fatalf("bulk xlsx export failed: %v\n%s", err, o)
	}

	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("xlsx is not a valid zip: %v", err)
	}
	defer zr.Close()

	var sheet string
	for _, file := range zr.File {
		if file.Name == "xl/worksheets/sheet1.xml" {
			rc, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open sheet: %v", err)
			}
			b, _ := io.ReadAll(rc)
			rc.Close()
			sheet = string(b)
		}
	}
	if sheet == "" {
		t.Fatalf("xlsx missing xl/worksheets/sheet1.xml")
	}
	for _, want := range []string{"rich@example.com", "mx1.example.com | mx2.example.com", "trust_grade"} {
		if !strings.Contains(sheet, want) {
			t.Fatalf("expected sheet to contain %q", want)
		}
	}
}

func TestJobsDownloadXLSXValidOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/validity-jobs/download" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("status"); got != "valid" {
			t.Fatalf("expected status=valid query, got %q", got)
		}
		if got := r.URL.Query().Get("format"); got != "json" {
			t.Fatalf("expected format=json query for xlsx conversion, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"results":[`+enrichedResult+`]}`)
	}))
	defer server.Close()

	out := filepath.Join(t.TempDir(), "job-results.xlsx")
	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}

	if o, err := runCLI(t, env, "jobs", "download", "job_123", "--format", "xlsx", "--valid-only", "--out", out); err != nil {
		t.Fatalf("jobs xlsx download failed: %v\n%s", err, o)
	}

	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("xlsx is not a valid zip: %v", err)
	}
	defer zr.Close()

	var sheet string
	for _, file := range zr.File {
		if file.Name == "xl/worksheets/sheet1.xml" {
			rc, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open sheet: %v", err)
			}
			b, _ := io.ReadAll(rc)
			rc.Close()
			sheet = string(b)
		}
	}
	if sheet == "" {
		t.Fatalf("xlsx missing xl/worksheets/sheet1.xml")
	}
	for _, want := range []string{"rich@example.com", "status", "reason"} {
		if !strings.Contains(sheet, want) {
			t.Fatalf("expected sheet to contain %q", want)
		}
	}
}

func TestJobsDownloadCSVValidOnlyCanonicalColumns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/validity-jobs/download" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("status"); got != "valid" {
			t.Fatalf("expected status=valid query, got %q", got)
		}
		if got := r.URL.Query().Get("format"); got != "json" {
			t.Fatalf("expected format=json query for local CSV flattening, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"results":[`+enrichedResult+`,`+bareResult+`]}`)
	}))
	defer server.Close()

	out := filepath.Join(t.TempDir(), "job-results.csv")
	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}

	if o, err := runCLI(t, env, "jobs", "download", "job_123", "--format", "csv", "--valid-only", "--out", out); err != nil {
		t.Fatalf("jobs csv download failed: %v\n%s", err, o)
	}

	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("failed to open csv: %v", err)
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse csv: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected header + 1 valid row, got %d rows", len(records))
	}

	header := records[0]
	if header[0] != "email" || header[1] != "status" || header[2] != "reason" {
		t.Fatalf("unexpected leading columns: %v", header[:3])
	}
	if colIndex(header, "mx") < 0 || colIndex(header, "trust_grade") < 0 || colIndex(header, "google_account") < 0 {
		t.Fatalf("expected canonical flattened columns, got: %v", header)
	}

	row := records[1]
	if row[colIndex(header, "email")] != "rich@example.com" {
		t.Fatalf("expected only valid row rich@example.com, got %q", row[colIndex(header, "email")])
	}
	if row[colIndex(header, "status")] != "valid" {
		t.Fatalf("status = %q, want valid", row[colIndex(header, "status")])
	}
	if row[colIndex(header, "mx")] != "mx1.example.com | mx2.example.com" {
		t.Fatalf("mx = %q", row[colIndex(header, "mx")])
	}
	if row[colIndex(header, "free_provider")] != "yes" {
		t.Fatalf("free_provider = %q, want yes", row[colIndex(header, "free_provider")])
	}
	if row[colIndex(header, "disposable")] != "no" {
		t.Fatalf("disposable = %q, want no", row[colIndex(header, "disposable")])
	}
}

func TestListsListJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agent/lists" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"id":"list_1","name":"Prospects","email_count":2,"created_at":"2026-07-28T00:00:00Z"}]`)
	}))
	defer server.Close()

	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}
	out, err := runCLI(t, env, "lists", "list", "--json")
	if err != nil {
		t.Fatalf("lists list failed: %v\n%s", err, out)
	}
	for _, want := range []string{`"id": "list_1"`, `"name": "Prospects"`, `"email_count": 2`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestListsAddEmails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agent/lists/list_1/emails" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		emails, _ := body["emails"].([]interface{})
		if len(emails) != 2 {
			t.Fatalf("expected 2 emails, got %v", body["emails"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"added":2}`)
	}))
	defer server.Close()

	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}
	out, err := runCLI(t, env, "lists", "add", "list_1", "--emails", "one@example.com", "--emails", "two@example.com")
	if err != nil {
		t.Fatalf("lists add failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Added 2") {
		t.Fatalf("expected add confirmation, got:\n%s", out)
	}
}

func TestJobsBulkIdentityJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/identity-jobs" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		emails, _ := body["emails"].([]interface{})
		if len(emails) != 2 {
			t.Fatalf("expected 2 emails, got %v", body["emails"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"job_i1","status":"queued"}`)
	}))
	defer server.Close()

	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}
	out, err := runCLI(t, env, "jobs", "bulk_email_identity", "--emails", "one@example.com", "--emails", "two@example.com", "--json")
	if err != nil {
		t.Fatalf("bulk_email_identity failed: %v\n%s", err, out)
	}
	for _, want := range []string{`"id": "job_i1"`, `"status": "queued"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestJobsGetEmailJobResultsPasswordBreached(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/password-jobs/results" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("id") != "job_p1" || q.Get("page") != "2" || q.Get("page_size") != "100" || q.Get("breached") != "1" {
			t.Fatalf("unexpected query: %s", q.Encode())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"items":[{"sha1":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","found":true}],"page":2}`)
	}))
	defer server.Close()

	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}
	out, err := runCLI(t, env, "jobs", "get_email_job_results", "job_p1", "--job-type", "password", "--page", "2", "--page-size", "100", "--breached", "--json")
	if err != nil {
		t.Fatalf("get_email_job_results failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"sha1": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"`) {
		t.Fatalf("expected password results json, got:\n%s", out)
	}
}

func TestJobsRetryIdentityJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/identity-jobs/retry" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("id") != "job_i1" {
			t.Fatalf("unexpected id query: %s", r.URL.Query().Encode())
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"requeued":3}`)
	}))
	defer server.Close()

	env := map[string]string{"ENCRATA_API_KEY": "test-key", "ENCRATA_BASE_URL": server.URL}
	out, err := runCLI(t, env, "jobs", "retry_email_job", "job_i1", "--job-type", "identity", "--json")
	if err != nil {
		t.Fatalf("retry_email_job failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"requeued": 3`) {
		t.Fatalf("expected retry response, got:\n%s", out)
	}
}
