package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Encratahq/cli/internal/output"
)

// readLines reads non-empty trimmed lines from a file.
func readLines(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(string(raw), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines, nil
}

// readFileBytes reads the raw contents of a file.
func readFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// writeFileBytes writes raw bytes to a file, creating parent directories as
// needed.
func writeFileBytes(path string, data []byte) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

// prettyJSON returns the raw payload pretty-printed with a 2-space indent,
// falling back to the original bytes if it is not valid JSON.
func prettyJSON(data json.RawMessage) []byte {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return data
	}
	return buf.Bytes()
}

// saveResult writes the full JSON payload to out and prints the absolute saved
// path to STDERR (keeping STDOUT pure for --json piping).
func saveResult(out string, data json.RawMessage) error {
	if err := writeFileBytes(out, prettyJSON(data)); err != nil {
		return err
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		abs = out
	}
	output.SavedPath(abs)
	return nil
}
