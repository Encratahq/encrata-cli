package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Error is an API-level failure carrying the HTTP status code.
type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("request failed with status %d", e.StatusCode)
}

func parseError(status int, data []byte) error {
	var body struct {
		Error   interface{} `json:"error"`
		Message interface{} `json:"message"`
		Detail  interface{} `json:"detail"`
		Details interface{} `json:"details"`
		Errors  interface{} `json:"errors"`
	}
	msg := ""
	if json.Unmarshal(data, &body) == nil {
		msg = firstErrorText(body.Message, body.Error, body.Detail, body.Details, body.Errors)
	}

	if msg == "" {
		switch status {
		case http.StatusUnauthorized:
			msg = "authentication failed — check your API key"
		case http.StatusPaymentRequired:
			msg = "insufficient credits"
		case http.StatusBadRequest:
			msg = "invalid request"
		case http.StatusTooManyRequests:
			msg = "rate limited — please wait and try again"
		default:
			msg = fmt.Sprintf("request failed with status %d", status)
		}
	}

	return &Error{StatusCode: status, Message: msg}
}

func firstErrorText(values ...interface{}) string {
	for _, value := range values {
		if text := errorText(value); text != "" {
			return text
		}
	}
	return ""
}

func errorText(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []interface{}:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if text := errorText(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "; ")
	case map[string]interface{}:
		for _, key := range []string{"message", "msg", "error", "detail", "field"} {
			if text := errorText(v[key]); text != "" {
				return text
			}
		}
		parts := make([]string, 0, len(v))
		for key, item := range v {
			if text := errorText(item); text != "" {
				parts = append(parts, fmt.Sprintf("%s: %s", key, text))
			}
		}
		return strings.Join(parts, "; ")
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}
