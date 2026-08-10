package cmd

import (
	"fmt"
	"strings"
)

// yesNo maps a boolField result ("true"/"false"/"") to yes/no/"".
func yesNo(b string) string {
	switch b {
	case "true":
		return "yes"
	case "false":
		return "no"
	default:
		return ""
	}
}

// pipeList joins the first array value found across keys with " | ", or returns
// the first non-empty scalar.
func pipeList(r map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		v, ok := lookupRaw(r, k)
		if !ok {
			continue
		}
		if arr, ok := v.([]interface{}); ok {
			parts := make([]string, 0, len(arr))
			for _, item := range arr {
				if item == nil {
					continue
				}
				if s := strings.TrimSpace(fmt.Sprintf("%v", item)); s != "" {
					parts = append(parts, s)
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, " | ")
			}
			continue
		}
		if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" {
			return s
		}
	}
	return ""
}

// truthy reports whether a decoded JSON value is a meaningful "present" value.
func truthy(v interface{}) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		s := strings.TrimSpace(t)
		return s != "" && !strings.EqualFold(s, "false")
	case float64:
		return t != 0
	default:
		return true
	}
}

// flagIfPresent renders yes/no when the presence object exists, else "".
// yes when any of keys resolves to a truthy value; otherwise no.
func flagIfPresent(r map[string]interface{}, presence string, keys ...string) string {
	if getMap(r, presence) == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := lookupRaw(r, k); ok && truthy(v) {
			return "yes"
		}
	}
	return "no"
}
