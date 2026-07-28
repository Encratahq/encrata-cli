package unit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/password"
)

// TestHashUpperHexSHA1 verifies the local hash is the UPPER-CASE hex SHA-1 and
// matches the canonical HIBP vector for "password".
func TestHashUpperHexSHA1(t *testing.T) {
	got := password.Hash([]byte("password"))
	want := "5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8"
	if got != want {
		t.Fatalf("Hash(\"password\") = %q, want %q", got, want)
	}
}

func TestZeroWipesBuffer(t *testing.T) {
	buf := []byte("secret")
	password.Zero(buf)
	for i, b := range buf {
		if b != 0 {
			t.Fatalf("byte %d not zeroed: %d", i, b)
		}
	}
}

// TestPrepareHashesDedupesFirstSeen verifies de-duplication preserves first-seen
// order and that plaintext buffers are wiped.
func TestPrepareHashesDedupesFirstSeen(t *testing.T) {
	lines := [][]byte{
		[]byte("alpha"),
		[]byte("beta"),
		[]byte("alpha"), // duplicate
		[]byte("gamma"),
		[]byte("beta"), // duplicate
	}
	hashes := password.PrepareHashes(lines)

	want := []string{
		password.Hash([]byte("alpha")),
		password.Hash([]byte("beta")),
		password.Hash([]byte("gamma")),
	}
	if len(hashes) != len(want) {
		t.Fatalf("expected %d unique hashes, got %d: %v", len(want), len(hashes), hashes)
	}
	for i := range want {
		if hashes[i] != want[i] {
			t.Fatalf("hash %d = %q, want %q", i, hashes[i], want[i])
		}
	}
	// Plaintext buffers must be wiped.
	for i, line := range lines {
		for _, b := range line {
			if b != 0 {
				t.Fatalf("line %d not zeroed after PrepareHashes", i)
			}
		}
	}
}

// TestBulkCapEnforceable checks that unique counts above MaxBulk are detectable,
// which is how the CLI rejects oversized batches before hitting the API.
func TestBulkCapEnforceable(t *testing.T) {
	if password.MaxBulk != 1000 {
		t.Fatalf("MaxBulk = %d, want 1000", password.MaxBulk)
	}
	lines := make([][]byte, 0, password.MaxBulk+1)
	for i := 0; i < password.MaxBulk+1; i++ {
		lines = append(lines, []byte("pw-"+string(rune('a'+i%26))+string(rune('0'+i/26%10))+intToStr(i)))
	}
	hashes := password.PrepareHashes(lines)
	if len(hashes) <= password.MaxBulk {
		t.Fatalf("expected more than %d unique hashes, got %d", password.MaxBulk, len(hashes))
	}
}

func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func TestSplitLinesTrimsAndSkipsEmpty(t *testing.T) {
	raw := []byte("  alpha  \r\n\n\tbeta\n   \ngamma")
	lines := password.SplitLines(raw)
	want := []string{"alpha", "beta", "gamma"}
	if len(lines) != len(want) {
		t.Fatalf("expected %d lines, got %d", len(want), len(lines))
	}
	for i := range want {
		if string(lines[i]) != want[i] {
			t.Fatalf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestCommas(t *testing.T) {
	cases := map[int]string{
		0:       "0",
		42:      "42",
		999:     "999",
		1000:    "1,000",
		12345:   "12,345",
		1000000: "1,000,000",
		-12345:  "-12,345",
	}
	for in, want := range cases {
		if got := password.Commas(in); got != want {
			t.Fatalf("Commas(%d) = %q, want %q", in, got, want)
		}
	}
}

// TestSingleResultBreachDecision verifies the exit-code decision for single checks.
func TestSingleResultBreachDecision(t *testing.T) {
	breached := password.SingleResult{Found: true, Count: 12345}
	if !breached.Breach() {
		t.Fatalf("expected Breach() true when found")
	}
	safe := password.SingleResult{Found: false}
	if safe.Breach() {
		t.Fatalf("expected Breach() false when not found")
	}
}

// TestBulkResultBreachDecision verifies the exit-code decision for bulk checks.
func TestBulkResultBreachDecision(t *testing.T) {
	if !(password.BulkResult{Breached: 1}).Breach() {
		t.Fatalf("expected Breach() true when breached > 0")
	}
	if (password.BulkResult{Breached: 0}).Breach() {
		t.Fatalf("expected Breach() false when breached == 0")
	}
}

// TestPasswordBreachesSendsHashOnly verifies the CLI transmits only the SHA-1
// hash — never the plaintext — and parses the single-check response.
func TestPasswordBreachesSendsHashOnly(t *testing.T) {
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
		if body["sha1"] != "5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8" {
			t.Fatalf("expected sha1 hash in body, got %v", body["sha1"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"prefix":"5BAA6","found":true,"count":12345,"message":"unsafe","credits":1}`))
	}))
	defer server.Close()

	client := api.New(server.URL, "test-key")
	data, err := client.PasswordBreaches(context.Background(), password.Hash([]byte("password")))
	if err != nil {
		t.Fatalf("password breaches failed: %v", err)
	}

	var res password.SingleResult
	if err := json.Unmarshal(data, &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !res.Found || res.Count != 12345 || res.Credits != 1 || res.Prefix != "5BAA6" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

// TestPasswordBreachesBulkSendsHashes verifies the bulk endpoint receives the
// sha1s array (not plaintext) and parses the bulk response.
func TestPasswordBreachesBulkSendsHashes(t *testing.T) {
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
		hashes, ok := body["sha1s"].([]interface{})
		if !ok || len(hashes) != 2 {
			t.Fatalf("expected 2 sha1s, got %v", body["sha1s"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":2,"breached":1,"results":[{"prefix":"ABCDE","found":true,"count":42},{"prefix":"F0A21","found":false,"count":0}],"credits":2}`))
	}))
	defer server.Close()

	client := api.New(server.URL, "test-key")
	hashes := password.PrepareHashes([][]byte{[]byte("alpha"), []byte("beta")})
	data, err := client.PasswordBreachesBulk(context.Background(), hashes)
	if err != nil {
		t.Fatalf("bulk password breaches failed: %v", err)
	}

	var res password.BulkResult
	if err := json.Unmarshal(data, &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.Total != 2 || res.Breached != 1 || res.Credits != 2 || len(res.Results) != 2 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if !res.Breach() {
		t.Fatalf("expected Breach() true")
	}
}

// TestPasswordBreachErrorStatuses verifies credit/gateway/service errors surface
// with the right status code and message.
func TestPasswordBreachErrorStatuses(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"payment required", http.StatusPaymentRequired, `{"error":"Insufficient credits. Please top up your account."}`},
		{"bad gateway", http.StatusBadGateway, `{"error":"Password breach lookup failed. Please try again."}`},
		{"service unavailable", http.StatusServiceUnavailable, `{"error":"Password breach lookup is temporarily unavailable"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := api.New(server.URL, "test-key")
			_, err := client.PasswordBreaches(context.Background(), password.Hash([]byte("x")))
			if err == nil {
				t.Fatalf("expected error for status %d", tc.status)
			}
			apiErr, ok := err.(*api.Error)
			if !ok {
				t.Fatalf("expected *api.Error, got %T: %v", err, err)
			}
			if apiErr.StatusCode != tc.status {
				t.Fatalf("expected status %d, got %d", tc.status, apiErr.StatusCode)
			}
			if apiErr.Message == "" {
				t.Fatalf("expected a non-empty error message")
			}
		})
	}
}
