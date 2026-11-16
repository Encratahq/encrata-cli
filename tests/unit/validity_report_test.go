package unit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Encratahq/cli/internal/api"
)

// The email service wraps validity in {success, result, message} (ENC-2763); the
// client hands back the bare report with the server's message folded in.
func TestEmailValidityUnwrapsTheEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":{"email":"a@b.com","status":"unknown","validity":"catch-all",` +
			`"conclusion":{"is_catch_all":true},"credits":1},"message":"Domain accepts all addresses."}`))
	}))
	defer server.Close()

	data, err := api.New(server.URL, "k").EmailValidity(context.Background(), "a@b.com")
	if err != nil {
		t.Fatalf("validity: %v", err)
	}
	var r map[string]interface{}
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatalf("not JSON: %s", data)
	}
	if r["email"] != "a@b.com" || r["validity"] != "catch-all" || r["credits"] != float64(1) {
		t.Fatalf("report not unwrapped: %v", r)
	}
	if r["message"] != "Domain accepts all addresses." {
		t.Fatalf("message not folded in: %v", r["message"])
	}
	if _, wrapped := r["success"]; wrapped {
		t.Fatalf("envelope leaked: %v", r)
	}
	if !strings.HasPrefix(string(data), `{"email":"a@b.com"`) {
		t.Fatalf("field order changed: %s", data)
	}
}

// A report with its own message keeps it.
func TestEmailValidityKeepsTheReportMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"result":{"email":"a@b.com","message":"own"},"message":"envelope"}`))
	}))
	defer server.Close()
	data, _ := api.New(server.URL, "k").EmailValidity(context.Background(), "a@b.com")
	if string(data) != `{"email":"a@b.com","message":"own"}` {
		t.Fatalf("got %s", data)
	}
}

// A failure envelope surfaces its message as the error.
func TestEmailValidityFailureEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"success":false,"result":{"code":"insufficient_credits"},"message":"You don't have enough credits."}`))
	}))
	defer server.Close()
	_, err := api.New(server.URL, "k").EmailValidity(context.Background(), "a@b.com")
	if err == nil || err.Error() != "You don't have enough credits." {
		t.Fatalf("err = %v", err)
	}
}

// A validity job is created through the typed bulk entrypoint (the per-kind
// create answers 405), and the enveloped job parses.
func TestCreateValidityJobFileUsesTheBulkEntrypoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/cli/jobs/bulk" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if typ := r.FormValue("type"); typ != "validity" {
			t.Fatalf("type = %q", typ)
		}
		f, h, err := r.FormFile("file")
		if err != nil || h.Filename != "list.csv" {
			t.Fatalf("file: %v %v", h, err)
		}
		body, _ := io.ReadAll(f)
		if string(body) != "a@b.com\n" {
			t.Fatalf("body = %q", body)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"success":true,"result":{"id":"j-1","status":"queued","total_emails":1},"message":""}`))
	}))
	defer server.Close()

	data, err := api.New(server.URL, "k").CreateValidityJobFile(context.Background(), "list.csv", []byte("a@b.com\n"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	job, err := api.ParseJob(data)
	if err != nil || job.ID != "j-1" || job.Status != "queued" || job.TotalEmails != 1 {
		t.Fatalf("job = %+v err = %v", job, err)
	}
}

// Results are cursor-paged on the by-id route: page 2 walks the cursor, the
// status filter is forwarded, and each row's report is folded in with the row's
// own verdict winning.
func TestGetValidityJobResultsWalksTheCursor(t *testing.T) {
	var cursors []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/api/cli/jobs/j-1" || q.Get("type") != "validity" || q.Get("status") != "invalid" {
			t.Fatalf("unexpected %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		cursors = append(cursors, q.Get("cursor"))
		if q.Get("cursor") == "0" {
			_, _ = w.Write([]byte(`{"success":true,"result":{"items":[{"email":"a@x.com","status":"invalid"}],"next_cursor":"c1"}}`))
			return
		}
		report, _ := json.Marshal(`{"email":"b@x.com","status":"undeliverable","validity":"invalid","outcome":{"reason":"mailbox_not_found"}}`)
		_, _ = w.Write([]byte(`{"success":true,"result":{"items":[{"email":"b@x.com","status":"invalid","reason":"no mailbox","report":` +
			string(report) + `}],"next_cursor":""}}`))
	}))
	defer server.Close()

	data, err := api.New(server.URL, "k").GetValidityJobResults(context.Background(), "j-1", 2, "invalid")
	if err != nil {
		t.Fatalf("results: %v", err)
	}
	if strings.Join(cursors, ",") != "0,c1" {
		t.Fatalf("cursors = %v", cursors)
	}
	var page struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.Unmarshal(data, &page); err != nil || len(page.Results) != 1 {
		t.Fatalf("page = %s err = %v", data, err)
	}
	row := page.Results[0]
	if row["email"] != "b@x.com" || row["status"] != "invalid" || row["reason"] != "no mailbox" ||
		row["report_status"] != "undeliverable" || row["validity"] != "invalid" {
		t.Fatalf("row = %v", row)
	}
	if o, _ := row["outcome"].(map[string]interface{}); o["reason"] != "mailbox_not_found" {
		t.Fatalf("report not folded in: %v", row)
	}
	if _, left := row["report"]; left {
		t.Fatalf("report string left on the row: %v", row)
	}
}

// Asking past the last page returns an empty page, not an error.
func TestGetValidityJobResultsPastTheEnd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"result":{"items":[{"email":"a@x.com","status":"valid"}],"next_cursor":""}}`))
	}))
	defer server.Close()
	data, err := api.New(server.URL, "k").GetValidityJobResults(context.Background(), "j-1", 3, "")
	if err != nil || !strings.Contains(string(data), `"results":[]`) {
		t.Fatalf("data = %s err = %v", data, err)
	}
}

// Cancel, retry and download address the job by id; the per-kind routes are gone.
func TestValidityJobActionsUseTheByIDRoutes(t *testing.T) {
	var got []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"success":true,"result":{"requeued":2}}`))
	}))
	defer server.Close()
	c := api.New(server.URL, "k")
	ctx := context.Background()
	if _, err := c.CancelValidityJob(ctx, "j-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.RetryValidityJob(ctx, "j-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DownloadValidityJob(ctx, "j-1", "valid", "csv"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"POST /api/cli/jobs/j-1/cancel?",
		"POST /api/cli/jobs/j-1/retry?",
		"GET /api/cli/jobs/j-1/download?format=csv&status=valid",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v", got)
	}
}

func TestParseJobReadsEveryShape(t *testing.T) {
	for name, body := range map[string]string{
		"enveloped": `{"success":true,"result":{"id":"j","status":"processing","valid_count":3},"message":""}`,
		"nested":    `{"job":{"id":"j","status":"processing","valid_count":3}}`,
		"bare":      `{"id":"j","status":"processing","valid_count":3}`,
	} {
		job, err := api.ParseJob(json.RawMessage(body))
		if err != nil || job.ID != "j" || job.Status != "processing" || job.ValidCount != 3 {
			t.Errorf("%s: job = %+v err = %v", name, job, err)
		}
	}
	if !api.JobIsTerminal("partially_done") {
		t.Error("partially_done is a finished job")
	}
}
