package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/textutil"
)

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
