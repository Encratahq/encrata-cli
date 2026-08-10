package unit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Encratahq/cli/internal/api"
)

func TestBulkValiditySearchStreamsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agent/bulk-validity-search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")

		// A start event, two result events, then done.
		_, _ = w.Write([]byte("event: start\ndata: {\"total\":2}\n\n"))
		_, _ = w.Write([]byte("data: {\"email\":\"one@example.com\",\"validity\":\"valid\"}\n\n"))
		_, _ = w.Write([]byte("event: result\ndata: {\"email\":\"two@example.com\",\"validity\":\"invalid\"}\n\n"))
		_, _ = w.Write([]byte("event: done\ndata: {\"processed\":2}\n\n"))
	}))
	defer server.Close()

	client := api.New(server.URL, "test-key")

	var emails []string
	var sawDone bool
	err := client.BulkValiditySearch(context.Background(), []string{"one@example.com", "two@example.com"}, "list.csv", func(ev api.BulkEvent) error {
		switch ev.Type {
		case "result":
			var res struct {
				Email string `json:"email"`
			}
			if err := json.Unmarshal(ev.Data, &res); err != nil {
				return err
			}
			emails = append(emails, res.Email)
		case "done":
			sawDone = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("bulk validity search failed: %v", err)
	}

	want := []string{"one@example.com", "two@example.com"}
	if len(emails) != len(want) {
		t.Fatalf("expected %d results, got %d: %v", len(want), len(emails), emails)
	}
	for i := range want {
		if emails[i] != want[i] {
			t.Fatalf("expected result %d to be %q, got %q", i, want[i], emails[i])
		}
	}
	if !sawDone {
		t.Fatalf("expected a done event")
	}
}

func TestEmailValidityPostsEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agent/email-validity" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body.Email != "user@example.com" {
			t.Fatalf("expected email in body, got %q", body.Email)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"email":"user@example.com","validity":"valid","credits":1}`))
	}))
	defer server.Close()

	client := api.New(server.URL, "test-key")
	data, err := client.EmailValidity(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("email validity failed: %v", err)
	}

	var res struct {
		Validity string `json:"validity"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.Validity != "valid" {
		t.Fatalf("expected validity 'valid', got %q", res.Validity)
	}
}
