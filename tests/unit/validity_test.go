package unit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Encratahq/cli/internal/api"
)

func TestEmailValidityPostsEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/email/validity" {
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
