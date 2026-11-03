package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestReadSSE(t *testing.T) {
	body := ": connected\n\n" +
		"event: job.completed\ndata: {\"job_id\":\"a\"}\n\n" +
		": ping\n\n" +
		"event: job.failed\r\ndata: {\"job_id\":\r\ndata: \"b\"}\r\n\r\n" +
		"data: no-event\n\n"
	type frame struct{ event, data string }
	var frames []frame
	lines := 0
	err := readSSE(strings.NewReader(body), func() { lines++ }, func(event, data string) error {
		frames = append(frames, frame{event, data})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []frame{
		{"job.completed", `{"job_id":"a"}`},
		{"job.failed", "{\"job_id\":\n\"b\"}"},
		{"", "no-event"},
	}
	if fmt.Sprint(frames) != fmt.Sprint(want) {
		t.Fatalf("frames = %q, want %q", frames, want)
	}
	if lines == 0 {
		t.Fatal("onLine never fired")
	}
}

func TestDecodeJobEvent(t *testing.T) {
	ev, ok := decodeJobEvent("job.completed", `{"job_id":"job_1","kind":"validity","status":"completed","processed":10,"total":10,"counts":{"valid":8,"invalid":2}}`)
	if !ok || ev.Event != EventJobCompleted || ev.JobID != "job_1" || ev.Counts["valid"] != 8 || !ev.Succeeded() {
		t.Fatalf("unexpected decode: %+v ok=%v", ev, ok)
	}
	for _, c := range []struct{ event, data string }{
		{"job.progress", `{"job_id":"x"}`},
		{"job.completed", `not json`},
		{"job.completed", `{"kind":"validity"}`},
	} {
		if _, ok := decodeJobEvent(c.event, c.data); ok {
			t.Errorf("expected %q %q to be skipped", c.event, c.data)
		}
	}
}

// TestSubscribeReconnectsAndDedupes: the first connection sends a frame then
// drops; the reconnect replays it (at-least-once) plus a new one. The caller
// must see each job exactly once, and the key must be sent on every dial.
func TestSubscribeReconnectsAndDedupes(t *testing.T) {
	var dials atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != subscribePath || r.Header.Get("X-Api-Key") != "enc_test" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		fmt.Fprint(w, ": connected\n\n")
		fmt.Fprint(w, "event: job.completed\ndata: {\"job_id\":\"a\",\"kind\":\"breaches\"}\n\n")
		if dials.Add(1) == 1 {
			fl.Flush()
			return // drop the connection
		}
		fmt.Fprint(w, "event: job.failed\ndata: {\"job_id\":\"b\",\"kind\":\"breaches\"}\n\n")
		fl.Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := New(srv.URL, "enc_test")
	var got []string
	retries := 0
	err := c.Subscribe(context.Background(), SubscribeHooks{
		OnRetry: func(error, time.Duration) { retries++ },
	}, func(ev JobEvent) error {
		got = append(got, ev.Event+":"+ev.JobID)
		if ev.JobID == "b" {
			return ErrStopSubscribe
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if strings.Join(got, ",") != "job.completed:a,job.failed:b" {
		t.Fatalf("events = %v", got)
	}
	if retries != 1 || dials.Load() != 2 {
		t.Fatalf("expected one reconnect, got retries=%d dials=%d", retries, dials.Load())
	}
}

func TestSubscribeDoesNotRetryAuthFailure(t *testing.T) {
	var dials atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dials.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"unauthorized","message":"Invalid API key."}`)
	}))
	defer srv.Close()

	err := New(srv.URL, "enc_bad").Subscribe(context.Background(), SubscribeHooks{}, func(JobEvent) error { return nil })
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected a 401 *Error, got %T %v", err, err)
	}
	if dials.Load() != 1 {
		t.Fatalf("auth failure must not be retried, dialled %d times", dials.Load())
	}
}

func TestSubscribeReturnsCallbackError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: job.completed\ndata: {\"job_id\":\"a\"}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	boom := errors.New("boom")
	err := New(srv.URL, "enc_test").Subscribe(context.Background(), SubscribeHooks{}, func(JobEvent) error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("expected the callback's error, got %v", err)
	}
}

func TestSubscribeStopsOnContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, ": connected\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	connected := make(chan struct{})
	go func() { <-connected; cancel() }()
	err := New(srv.URL, "enc_test").Subscribe(ctx, SubscribeHooks{OnConnect: func() { close(connected) }}, func(JobEvent) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestScanMalwareRoute(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotPath, gotBody = r.Method+" "+r.URL.Path, string(b)
		fmt.Fprint(w, `{"success":true,"result":{"verdict":"clean"}}`)
	}))
	defer srv.Close()

	_, err := New(srv.URL, "enc_test").ScanMalware(context.Background(), MalwareScanRequest{Kind: MalwareKindURL, URL: "https://example.com", STIX: true})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "POST /api/cli/malware/scan" || gotBody != `{"kind":"url","url":"https://example.com","stix":true}` {
		t.Fatalf("request = %s %s", gotPath, gotBody)
	}
}
