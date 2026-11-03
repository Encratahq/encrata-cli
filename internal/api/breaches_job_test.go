package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestStreamCutIsClassified pins that a body cut off after the headers (here an
// abrupt close mid chunked-body, standing in for an HTTP/2 RST_STREAM from
// Envoy) surfaces as *StreamCutError, with the events before it delivered.
func TestStreamCutIsClassified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, buf, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		defer conn.Close()
		event := "data: {\"email\":\"a@acme.com\"}\n\n"
		fmt.Fprintf(buf, "HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\nTransfer-Encoding: chunked\r\n\r\n")
		fmt.Fprintf(buf, "%x\r\n%s\r\n", len(event), event)
		buf.Flush() // then close without the terminating chunk
	}))
	defer srv.Close()

	got := 0
	err := newTestClient(srv.URL).BulkBreachesSearch(context.Background(), []string{"a@acme.com", "b@acme.com"}, "", func(BulkEvent) error {
		got++
		return nil
	})
	var cut *StreamCutError
	if !errors.As(err, &cut) {
		t.Fatalf("expected *StreamCutError, got %T: %v", err, err)
	}
	if got != 1 {
		t.Fatalf("expected the 1 event sent before the cut, got %d", got)
	}
}

// TestStreamRejectionIsNotACut keeps up-front HTTP errors (e.g. 402) as *Error.
func TestStreamRejectionIsNotACut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error":"Insufficient credits. Need 10, have 2."}`))
	}))
	defer srv.Close()

	err := newTestClient(srv.URL).BulkBreachesSearch(context.Background(), []string{"a@acme.com"}, "", func(BulkEvent) error { return nil })
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("expected a 402 *Error, got %T: %v", err, err)
	}
	if apiErr.Message != "Insufficient credits. Need 10, have 2." {
		t.Fatalf("server message lost: %q", apiErr.Message)
	}
}

func TestParseBreachesJobEnvelopes(t *testing.T) {
	for name, body := range map[string]string{
		"bare":     `{"id":"bj_1","status":"processing","total_emails":8569,"processed_count":455,"breached_count":12,"credits_used":455}`,
		"job":      `{"job":{"id":"bj_1","status":"processing","total_emails":8569,"processed_count":455,"breached_count":12,"credits_used":455}}`,
		"envelope": `{"success":true,"result":{"id":"bj_1","status":"processing","total_emails":8569,"processed_count":455,"breached_count":12,"credits_used":455},"message":""}`,
	} {
		job, err := ParseBreachesJob(json.RawMessage(body))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if job.ID != "bj_1" || job.TotalEmails != 8569 || job.ProcessedCount != 455 || job.BreachedCount != 12 || job.CreditsUsed != 455 {
			t.Errorf("%s: decoded %+v", name, job)
		}
	}
}

func TestNextPollIntervalBacksOffToCap(t *testing.T) {
	d := breachesPollMin
	seen := []time.Duration{d}
	for i := 0; i < 5; i++ {
		d = nextPollInterval(d)
		seen = append(seen, d)
	}
	if seen[1] != 3*time.Second || seen[len(seen)-1] != breachesPollMax {
		t.Fatalf("unexpected backoff sequence %v", seen)
	}
}

func TestPollBreachesJobRoutesAndStopsWhenTerminal(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/jobs/breaches" || r.URL.Query().Get("id") != "bj_1" {
			t.Errorf("unexpected request %s", r.URL)
		}
		calls++
		_, _ = w.Write([]byte(`{"id":"bj_1","status":"completed","total_emails":3,"processed_count":3}`))
	}))
	defer srv.Close()

	job, err := newTestClient(srv.URL).PollBreachesJob(context.Background(), "bj_1", nil)
	if err != nil || job.Status != "completed" || calls != 1 {
		t.Fatalf("job=%+v err=%v calls=%d", job, err, calls)
	}
}

func TestFetchBreachesJobItemsFollowsCursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/api/cli/jobs/bj_1" || q.Get("type") != "breaches" || q.Get("breached") != "1" {
			t.Errorf("unexpected request %s", r.URL)
		}
		switch q.Get("cursor") {
		case "0":
			_, _ = w.Write([]byte(`{"items":[{"email":"a@acme.com","breached":true,"breach_count":1,"services":"Apollo","data":"{\"breach_count\":1,\"services\":[{\"name\":\"Apollo\"}]}"}],"next_cursor":"h1"}`))
		case "h1":
			_, _ = w.Write([]byte(`{"items":[{"email":"b@acme.com","breached":true,"breach_count":1}],"next_cursor":""}`))
		default:
			t.Errorf("unexpected cursor %q", q.Get("cursor"))
		}
	}))
	defer srv.Close()

	var items []BreachesJobItem
	err := newTestClient(srv.URL).FetchBreachesJobItems(context.Background(), "bj_1", true, func(page []BreachesJobItem) error {
		items = append(items, page...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Email != "a@acme.com" || items[1].Email != "b@acme.com" {
		t.Fatalf("unexpected items %+v", items)
	}
	var report struct {
		Services []struct{ Name string } `json:"services"`
	}
	if err := json.Unmarshal(items[0].Report, &report); err != nil || len(report.Services) != 1 || report.Services[0].Name != "Apollo" {
		t.Fatalf("embedded report not decoded: %s (%v)", items[0].Report, err)
	}
	if items[1].Report != nil {
		t.Fatalf("row without data should have no report, got %s", items[1].Report)
	}
}

// TestBreachesJobRoutes pins each breaches-job call to the jobs service's current
// routes: create through the typed bulk entrypoint, everything per-job by id.
func TestBreachesJobRoutes(t *testing.T) {
	type hit struct{ method, path, query, body string }
	var got hit
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = hit{r.Method, r.URL.Path, r.URL.RawQuery, string(b)}
		_, _ = w.Write([]byte(`{"items":[],"next_cursor":""}`))
	}))
	defer srv.Close()
	c := newTestClient(srv.URL)
	ctx := context.Background()

	cases := []struct {
		name string
		call func() error
		want hit
	}{
		{"create", func() error { _, err := c.CreateBreachesJob(ctx, []string{"a@acme.com"}, "list.csv"); return err },
			hit{"POST", "/api/cli/jobs/bulk", "", `{"emails":["a@acme.com"],"file_name":"list.csv","type":"breaches"}`}},
		{"status", func() error { _, err := c.GetBreachesJob(ctx, "bj_1"); return err },
			hit{"GET", "/api/cli/jobs/breaches", "id=bj_1", ""}},
		{"results", func() error { _, err := c.GetBreachesJobResults(ctx, "bj_1", 1, 50, true); return err },
			hit{"GET", "/api/cli/jobs/bj_1", "breached=1&cursor=0&page_size=50&type=breaches", ""}},
		{"download", func() error { _, err := c.DownloadBreachesJob(ctx, "bj_1", true, "csv"); return err },
			hit{"GET", "/api/cli/jobs/bj_1/download", "breached=1&format=csv", ""}},
		{"cancel", func() error { _, err := c.CancelBreachesJob(ctx, "bj_1"); return err },
			hit{"POST", "/api/cli/jobs/bj_1/cancel", "", ""}},
		{"retry", func() error { _, err := c.RetryBreachesJob(ctx, "bj_1"); return err },
			hit{"POST", "/api/cli/jobs/bj_1/retry", "", ""}},
	}
	for _, tc := range cases {
		got = hit{}
		if err := tc.call(); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("%s = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

// TestGetBreachesJobResultsWalksToPage: the results route is cursor-only, so a
// requested page N is reached by following next_cursor N-1 times.
func TestGetBreachesJobResultsWalksToPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("cursor") {
		case "0":
			_, _ = w.Write([]byte(`{"items":[{"email":"p1@acme.com"}],"next_cursor":"c2"}`))
		case "c2":
			_, _ = w.Write([]byte(`{"items":[{"email":"p2@acme.com"}],"next_cursor":""}`))
		default:
			t.Errorf("unexpected cursor %q", r.URL.Query().Get("cursor"))
		}
	}))
	defer srv.Close()
	c := newTestClient(srv.URL)

	data, err := c.GetBreachesJobResults(context.Background(), "bj_1", 2, 1, false)
	if err != nil || !strings.Contains(string(data), "p2@acme.com") {
		t.Fatalf("page 2 = %s, %v", data, err)
	}
	data, err = c.GetBreachesJobResults(context.Background(), "bj_1", 5, 1, false)
	if err != nil || string(data) != `{"items":[],"next_cursor":""}` {
		t.Fatalf("past the end = %s, %v", data, err)
	}
}
