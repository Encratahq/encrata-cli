package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// These tests pin the CLI's request shapes to the backend contracts documented
// in encrata/backend/internal/handlers/workflows.go and workflow_integrations.go.
// They assert byte-for-byte: paths, methods, the multipart file field name,
// the run body key (file_id), the Idempotency-Key header, the /sheet suffix,
// and that download follows the 302 to a presigned URL.

func newTestClient(baseURL string) *Client {
	c := New(baseURL, "enc_test")
	// Follow redirects (default) but keep timeouts small for tests.
	return c
}

func TestUploadWorkflowFile_MultipartFieldAndResponse(t *testing.T) {
	var gotPath, gotMethod, gotFileField, gotWorkflowID string
	var gotFileContent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		gotWorkflowID = r.FormValue("workflow_id")
		f, hdr, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("expected form field 'file': %v", err)
		}
		defer f.Close()
		gotFileField = hdr.Filename
		b, _ := io.ReadAll(f)
		gotFileContent = string(b)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"file_1","filename":"emails.csv","row_count":3,"identifier_type":"email","identifier_column":"email"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	resp, err := c.UploadWorkflowFile(context.Background(), "emails.csv", []byte("a@x.com\nb@y.com\n"), "wf_9")
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if gotPath != "/api/cli/workflows/files" || gotMethod != http.MethodPost {
		t.Fatalf("path/method = %s %s", gotMethod, gotPath)
	}
	if gotFileField != "emails.csv" || !strings.Contains(gotFileContent, "a@x.com") {
		t.Fatalf("file field name/content wrong: %q %q", gotFileField, gotFileContent)
	}
	if gotWorkflowID != "wf_9" {
		t.Fatalf("workflow_id = %q", gotWorkflowID)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(resp, &m); err != nil {
		t.Fatalf("resp json: %v", err)
	}
	for _, k := range []string{"id", "filename", "row_count", "identifier_type", "identifier_column"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing response field %q (backend contract)", k)
		}
	}
}

func TestRunWorkflow_BodyFileIDAndIdempotencyHeader(t *testing.T) {
	var gotPath, gotIdem string
	var body map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotIdem = r.Header.Get("Idempotency-Key")
		json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"run_1","workflow_id":"wf_9","status":"pending","trigger_type":"manual","created_at":"2026-07-31T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	if _, err := c.RunWorkflow(context.Background(), "wf_9", "file_1", "idem-123"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/api/cli/workflows/wf_9/run" {
		t.Fatalf("path = %q", gotPath)
	}
	if body["file_id"] != "file_1" {
		t.Fatalf("body file_id = %v (want file_1)", body["file_id"])
	}
	if gotIdem != "idem-123" {
		t.Fatalf("Idempotency-Key = %q", gotIdem)
	}
}

func TestGetWorkflowRun_PathAndWrapper(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"run":{"id":"run_1","status":"success","output_key":"k","output_rows":3},"steps":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	resp, err := c.GetWorkflowRun(context.Background(), "run_1")
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if gotPath != "/api/cli/workflows/runs/run_1" {
		t.Fatalf("path = %q", gotPath)
	}
	var wrap struct {
		Run   map[string]interface{} `json:"run"`
		Steps []interface{}          `json:"steps"`
	}
	if err := json.Unmarshal(resp, &wrap); err != nil {
		t.Fatalf("resp json: %v", err)
	}
	if _, ok := wrap.Run["output_rows"]; !ok {
		t.Fatalf("run.output_rows missing (backend contract)")
	}
}

func TestDownloadWorkflowRunOutput_Follows302(t *testing.T) {
	presigned := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte("email,validity\na@x.com,valid\n"))
	}))
	defer presigned.Close()

	var gotPath string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		http.Redirect(w, r, presigned.URL, http.StatusFound)
	}))
	defer api.Close()

	c := newTestClient(api.URL)
	data, err := c.DownloadWorkflowRunOutput(context.Background(), "run_1")
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if gotPath != "/api/cli/workflows/runs/run_1/download" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.Contains(string(data), "email,validity") {
		t.Fatalf("csv not streamed from presigned url: %q", string(data))
	}
}

func TestCreateIntegrationSheet_UsesSheetSuffix(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"spreadsheet_id":"s1","spreadsheet_url":"http://x","sheet_name":"Sheet1"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	if _, err := c.CreateIntegrationSheet(context.Background(), "int_1", "My Sheet"); err != nil {
		t.Fatalf("create sheet: %v", err)
	}
	// Backend route is /sheet, NOT /create-sheet.
	if gotPath != "/api/cli/workflows/integrations/int_1/sheet" {
		t.Fatalf("path = %q (must end in /sheet)", gotPath)
	}
}

func TestNangoSave_BodyFields(t *testing.T) {
	var body map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"int_1","provider_config_key":"google-sheet"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	if _, err := c.NangoSave(context.Background(), "conn_1", "google-sheet", "My Google"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if body["connection_id"] != "conn_1" || body["provider_config_key"] != "google-sheet" || body["label"] != "My Google" {
		t.Fatalf("save body wrong: %v", body)
	}
}
