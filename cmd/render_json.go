package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

// lookupRaw resolves a flat key or a dotted "parent.child" path.
func lookupRaw(r map[string]interface{}, key string) (interface{}, bool) {
	if strings.Contains(key, ".") {
		parts := strings.SplitN(key, ".", 2)
		if child, ok := r[parts[0]].(map[string]interface{}); ok {
			return lookupRaw(child, parts[1])
		}
		return nil, false
	}
	v, ok := r[key]
	if !ok || v == nil {
		return nil, false
	}
	return v, true
}

// field returns the first non-empty string value across the given keys.
func field(r map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := lookupRaw(r, k); ok {
			switch t := v.(type) {
			case float64:
				return formatNumber(t)
			default:
				s := strings.TrimSpace(fmt.Sprintf("%v", v))
				if s != "" && s != "<nil>" {
					return s
				}
			}
		}
	}
	return ""
}

// boolField renders a present boolean as true/false, or "" when absent.
func boolField(r map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := lookupRaw(r, k); ok {
			switch t := v.(type) {
			case bool:
				return strconv.FormatBool(t)
			case string:
				s := strings.ToLower(strings.TrimSpace(t))
				if s == "true" || s == "false" {
					return s
				}
			}
		}
	}
	return ""
}

// listField joins an array value with ", ", or returns a scalar as-is.
func listField(r map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := lookupRaw(r, k); ok {
			if arr, ok := v.([]interface{}); ok {
				if len(arr) == 0 {
					continue
				}
				return joinInterfaces(arr)
			}
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

// countField returns the count of a list, or a numeric value as an integer.
func countField(r map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := lookupRaw(r, k); ok {
			switch t := v.(type) {
			case []interface{}:
				return strconv.Itoa(len(t))
			case float64:
				return strconv.Itoa(int(t))
			case string:
				if t != "" {
					return t
				}
			}
		}
	}
	return ""
}

// getMap returns a nested object, or nil.
func getMap(r map[string]interface{}, key string) map[string]interface{} {
	if v, ok := r[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

// firstArr returns the first present array across the given keys.
func firstArr(r map[string]interface{}, keys ...string) []interface{} {
	for _, k := range keys {
		if v, ok := lookupRaw(r, k); ok {
			if arr, ok := v.([]interface{}); ok {
				return arr
			}
		}
	}
	return nil
}

func asMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func formatNumber(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// creditsValue returns the credit cost from a response, defaulting to "0".
func creditsValue(r map[string]interface{}) string {
	if v := field(r, "credits", "credits_used", "credit"); v != "" {
		return v
	}
	return "0"
}

// intOf parses a numeric-ish string, defaulting to 0.
func intOf(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
