package unit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Encratahq/cli/internal/api"
)

func TestBulkLookupAcceptsDefaultAndResultSSEEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agent/bulk-lookup" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")

		// Default SSE event with wrapped data.
		_, _ = w.Write([]byte("data: {\"data\":{\"email\":\"default@example.com\"}}\n\n"))

		// Explicit result event with wrapped data.
		_, _ = w.Write([]byte("event: result\ndata: {\"data\":{\"email\":\"result@example.com\"}}\n\n"))

		// Non-result events should be ignored.
		_, _ = w.Write([]byte("event: progress\ndata: {\"email\":\"ignored@example.com\"}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := api.New(server.URL, "test-key")
	var emails []string
	err := client.BulkLookup(context.Background(), []string{"a@example.com"}, nil, func(item json.RawMessage) error {
		var person struct {
			Email string `json:"email"`
		}
		if err := json.Unmarshal(item, &person); err != nil {
			return err
		}
		emails = append(emails, person.Email)
		return nil
	})
	if err != nil {
		t.Fatalf("bulk lookup failed: %v", err)
	}

	want := []string{"default@example.com", "result@example.com"}
	if len(emails) != len(want) {
		t.Fatalf("expected %d results, got %d: %v", len(want), len(emails), emails)
	}
	for i := range want {
		if emails[i] != want[i] {
			t.Fatalf("expected result %d to be %q, got %q", i, want[i], emails[i])
		}
	}
}

func TestBulkSearchAcceptsDefaultAndResultSSEEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/bulk-google-search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")

		// Both default and explicit result events should be collected.
		_, _ = w.Write([]byte("data: {\"query\":\"default\"}\n\n"))
		_, _ = w.Write([]byte("event: result\ndata: {\"query\":\"result\"}\n\n"))

		// Non-result events should be ignored.
		_, _ = w.Write([]byte("event: progress\ndata: {\"query\":\"ignored\"}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := api.New(server.URL, "test-key")
	data, err := client.BulkSearch(context.Background(), "/api/bulk-google-search", []string{"query"})
	if err != nil {
		t.Fatalf("bulk search failed: %v", err)
	}

	var results []struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(data, &results); err != nil {
		t.Fatalf("failed to decode results: %v", err)
	}

	want := []string{"default", "result"}
	if len(results) != len(want) {
		t.Fatalf("expected %d results, got %d: %v", len(want), len(results), results)
	}
	for i := range want {
		if results[i].Query != want[i] {
			t.Fatalf("expected result %d to be %q, got %q", i, want[i], results[i].Query)
		}
	}
}
