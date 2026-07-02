package cmd

import (
	"encoding/json"
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
func simpleGet(fn func(*api.Client, string) (json.RawMessage, error), title string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		data, err := fn(client, args[0])
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
