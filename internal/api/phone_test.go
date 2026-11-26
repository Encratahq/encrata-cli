package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	ok := map[string]string{
		"+14155552671":        "+14155552671",
		" +1 (415) 555-2671 ": "+14155552671",
		"+44 20 7946 0958":    "+442079460958",
		"0014155552671":       "+14155552671",
		"+49.30.1234.5678":    "+493012345678",
		"+33/1/23/45/67/89":   "+33123456789",
	}
	for in, want := range ok {
		got, err := NormalizePhone(in)
		if err != nil || got != want {
			t.Errorf("NormalizePhone(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "4155552671", "+0123456789", "+1415abc2671", "+123", "+1234567890123456", "jane@acme.com"} {
		if got, err := NormalizePhone(in); err == nil {
			t.Errorf("NormalizePhone(%q) = %q, want an error", in, got)
		}
	}
}

// phoneServer records each request and answers in the lookup envelope.
func phoneServer(t *testing.T, gotPath *string, gotBody *map[string]interface{}) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		*gotBody = map[string]interface{}{}
		_ = json.Unmarshal(b, gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"message":"Phone validity ready.","result":{"phone":"+14155552671","status":"valid","credits":1}}`)
	}))
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL, APIKey: "k", HTTPClient: srv.Client()}
}

func TestPhoneLookupRequestAndUnwrap(t *testing.T) {
	var path string
	var body map[string]interface{}
	c := phoneServer(t, &path, &body)

	data, err := c.PhoneLookup(context.Background(), PhoneFeatureValidity, "+14155552671")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/cli/phone/validity" || body["phone"] != "+14155552671" {
		t.Fatalf("request = %s %v", path, body)
	}
	var r map[string]interface{}
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	if r["status"] != "valid" || r["message"] != "Phone validity ready." {
		t.Fatalf("result not unwrapped: %s", data)
	}

	if _, err := c.PhoneLookup(context.Background(), "metered/search", "+14155552671"); err == nil {
		t.Fatal("unknown feature must be refused client-side")
	}
}

func TestPhoneSearchAndVoipRequests(t *testing.T) {
	var path string
	var body map[string]interface{}
	c := phoneServer(t, &path, &body)

	if _, err := c.PhoneSearch(context.Background(), "+14155552671", []string{"risk", "carrier"}); err != nil {
		t.Fatal(err)
	}
	inc, _ := body["include"].([]interface{})
	if path != "/api/cli/phone/search" || body["query"] != "+14155552671" || len(inc) != 2 {
		t.Fatalf("search request = %s %v", path, body)
	}

	if _, err := c.PhoneSearch(context.Background(), "+14155552671", nil); err != nil {
		t.Fatal(err)
	}
	if _, has := body["include"]; has {
		t.Fatalf("empty include must be omitted: %v", body)
	}

	if _, err := c.PhoneVoip(context.Background(), "+14155552671", "current"); err != nil {
		t.Fatal(err)
	}
	if path != "/api/cli/phone/voip" || body["phone"] != "+14155552671" || body["mode"] != "current" {
		t.Fatalf("voip request = %s %v", path, body)
	}
}
