package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/textutil"
	"github.com/spf13/cobra"
)

// newClient validates configuration and returns a ready API client.
func newClient() (*api.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return api.New(cfg.BaseURL, cfg.APIKey), nil
}

// jsonMode reports whether output should be raw JSON.
func jsonMode() bool {
	return cfg.Output == "json"
}

// decode unmarshals raw API data, falling back to printing JSON on failure.
func decode(data json.RawMessage, v interface{}) bool {
	if err := json.Unmarshal(data, v); err != nil {
		output.JSON(data)
		return false
	}
	return true
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func improveArgErrors(cmd *cobra.Command) {
	for _, child := range cmd.Commands() {
		improveArgErrors(child)
	}
	if cmd.Args == nil {
		return
	}

	original := cmd.Args
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		err := original(cmd, args)
		if err == nil || !isCobraArgError(err) {
			return err
		}
		return friendlyArgError(cmd, args)
	}
}

func isCobraArgError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "accepts ") ||
		strings.Contains(message, "requires at least ") ||
		strings.Contains(message, "requires no arguments")
}

func friendlyArgError(cmd *cobra.Command, args []string) error {
	usage := strings.TrimSpace(cmd.UseLine())
	if usage == "" {
		usage = cmd.CommandPath()
	}
	usage = themedUsage(usage, cmd.CommandPath())
	help := fmt.Sprintf("%s %s", output.Dim.Sprint("Try"), output.Accent.Sprintf("%s --help", cmd.CommandPath()))

	if len(args) == 0 {
		return fmt.Errorf("%s\n\n%s\n  %s\n\n%s", output.Bold.Sprintf("missing %s", requiredInputName(cmd)), output.Dim.Sprint("Usage"), usage, help)
	}

	return fmt.Errorf("%s\n\n%s\n  %s\n\n%s", output.Bold.Sprint("wrong input format"), output.Dim.Sprint("Usage"), usage, help)
}

func friendlyFormatError(cmd *cobra.Command, message string) error {
	usage := strings.TrimSpace(cmd.UseLine())
	if usage == "" {
		usage = cmd.CommandPath()
	}
	usage = themedUsage(usage, cmd.CommandPath())
	help := fmt.Sprintf("%s %s", output.Dim.Sprint("Try"), output.Accent.Sprintf("%s --help", cmd.CommandPath()))

	return fmt.Errorf("%s\n\n%s\n  %s\n\n%s", output.Bold.Sprint(message), output.Dim.Sprint("Format"), usage, help)
}

func themedUsage(usage, commandPath string) string {
	if commandPath == "" {
		return usage
	}
	return strings.Replace(usage, commandPath, output.Brand.Sprint(commandPath), 1)
}

func requiredInputName(cmd *cobra.Command) string {
	for _, field := range strings.Fields(cmd.Use) {
		if strings.HasPrefix(field, "[") && strings.HasSuffix(field, "]") {
			name := strings.Trim(field, "[]")
			name = strings.TrimSuffix(name, "...")
			if name != "" {
				return name
			}
		}
	}
	return "required input"
}

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

func getArr(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key].([]interface{}); ok {
		return v
	}
	return nil
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func getStr(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getNestedStr(m map[string]interface{}, parent, key string) string {
	if child, ok := m[parent].(map[string]interface{}); ok {
		return getStr(child, key)
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func truncate(s string, max int) string {
	return textutil.Truncate(s, max)
}

// printNonEmptyKV prints only the key/value pairs whose value is non-empty.
// Returns true if it printed anything.
func printNonEmptyKV(pairs ...string) bool {
	filtered := make([]string, 0, len(pairs))
	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i+1] == "" {
			continue
		}
		filtered = append(filtered, pairs[i], pairs[i+1])
	}
	if len(filtered) == 0 {
		return false
	}
	output.KV(filtered...)
	return true
}

func joinInterfaces(values []interface{}) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if value != nil {
			parts = append(parts, fmt.Sprintf("%v", value))
		}
	}
	return strings.Join(parts, ", ")
}

// unwrapArray returns a JSON array whether it is bare or nested under key.
func unwrapArray(data json.RawMessage, key string) []interface{} {
	var arr []interface{}
	if json.Unmarshal(data, &arr) == nil {
		return arr
	}
	var obj map[string]interface{}
	if json.Unmarshal(data, &obj) == nil {
		if v, ok := obj[key].([]interface{}); ok {
			return v
		}
	}
	return nil
}

// simpleGet builds a RunE that fetches a resource by ID and prints it as JSON.
func simpleGet(fn func(*api.Client, context.Context, string) (json.RawMessage, error), title string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		spinner := startSpinner("Loading details...")
		data, err := fn(client, cmd.Context(), args[0])
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}
		if !jsonMode() {
			output.Header(title + ": " + args[0])
		}
		output.JSON(data)
		return nil
	}
}
