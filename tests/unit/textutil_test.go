package unit

import (
	"testing"
	"unicode/utf8"

	"github.com/Encratahq/cli/internal/textutil"
)

func TestTruncateKeepsUTF8Valid(t *testing.T) {
	// Truncate should count runes, not bytes.
	got := textutil.Truncate("ab🙂cd", 3)
	want := "ab🙂..."
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	// The result must remain valid UTF-8.
	if !utf8.ValidString(got) {
		t.Fatalf("expected valid UTF-8, got %q", got)
	}
}

func TestTruncateSmallLimits(t *testing.T) {
	// Zero or negative limits return an empty string.
	if got := textutil.Truncate("hello", 0); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}

	// Short strings are returned unchanged.
	if got := textutil.Truncate("hello", 10); got != "hello" {
		t.Fatalf("expected original string, got %q", got)
	}
}
