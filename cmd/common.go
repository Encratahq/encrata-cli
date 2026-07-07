package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
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
