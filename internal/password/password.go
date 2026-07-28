// Package password holds the local, network-free logic behind the
// `encrata password` command: SHA-1 hashing, de-duplication, request-size
// limits, response parsing and verdict formatting.
//
// Privacy: plaintext passwords never leave this package. Callers hash locally
// and only ever transmit the resulting UPPER-CASE hex SHA-1 digest. Nothing
// here logs, caches, prints or persists plaintext.
package password

import (
	"crypto/sha1"
	"encoding/hex"
	"strconv"
	"strings"
)

// MaxBulk is the maximum number of unique hashes accepted per bulk request.
// The API rejects anything larger.
const MaxBulk = 1000

// Hash returns the UPPER-CASE hex SHA-1 digest of the given plaintext bytes.
// It does not modify or retain the input; callers should Zero the buffer once
// they no longer need it.
func Hash(plaintext []byte) string {
	sum := sha1.Sum(plaintext)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// Zero overwrites a plaintext buffer in place so it no longer lingers in memory.
func Zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// PrepareHashes hashes each plaintext line, de-duplicates the resulting digests
// in first-seen order, and zeroes every plaintext buffer as it goes. The
// plaintext slices are consumed (wiped) and must not be reused by the caller.
func PrepareHashes(lines [][]byte) []string {
	seen := make(map[string]bool, len(lines))
	hashes := make([]string, 0, len(lines))
	for _, line := range lines {
		h := Hash(line)
		Zero(line)
		if h == "" || seen[h] {
			continue
		}
		seen[h] = true
		hashes = append(hashes, h)
	}
	return hashes
}

// SplitLines breaks raw input into trimmed, non-empty plaintext byte slices,
// one per line. Each returned slice is a fresh copy so the original buffer can
// be zeroed independently.
func SplitLines(raw []byte) [][]byte {
	var lines [][]byte
	for _, part := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimRight(strings.TrimSuffix(part, "\r"), " \t")
		trimmed = strings.TrimLeft(trimmed, " \t")
		if trimmed == "" {
			continue
		}
		lines = append(lines, []byte(trimmed))
	}
	return lines
}

// SingleResult is the parsed response of a single password breach check.
type SingleResult struct {
	Prefix  string `json:"prefix"`
	Found   bool   `json:"found"`
	Count   int    `json:"count"`
	Message string `json:"message"`
	Credits int    `json:"credits"`
}

// Breach reports whether the checked password appeared in a known breach.
func (r SingleResult) Breach() bool { return r.Found }

// BulkEntry is one hash's result inside a bulk response.
type BulkEntry struct {
	Prefix string `json:"prefix"`
	Found  bool   `json:"found"`
	Count  int    `json:"count"`
}

// BulkResult is the parsed response of a bulk password breach check.
type BulkResult struct {
	Total    int         `json:"total"`
	Breached int         `json:"breached"`
	Results  []BulkEntry `json:"results"`
	Credits  int         `json:"credits"`
}

// Breach reports whether at least one checked password appeared in a breach.
func (r BulkResult) Breach() bool { return r.Breached > 0 }

// Commas formats an integer with thousands separators, e.g. 12345 -> "12,345".
func Commas(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	lead := len(s) % 3
	if lead > 0 {
		b.WriteString(s[:lead])
	}
	for i := lead; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
