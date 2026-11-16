package api

import (
	"bytes"
	"encoding/json"
	"strings"
)

// UnwrapEnvelope returns the payload of a {"job": {...}} or
// {"success": ..., "result": {...}} response, or data unchanged when bare.
func UnwrapEnvelope(data json.RawMessage) json.RawMessage { return unwrapEnvelope(data) }

// unwrapResult returns the payload of a {success, result, message} response, with
// the envelope's message folded in when the payload has none of its own, so a
// renderer reading the payload still sees the server's sentence. A bare body (an
// older server, or a route that never wrapped) is returned unchanged. Field order
// of the payload is preserved.
func unwrapResult(data json.RawMessage) json.RawMessage {
	var env struct {
		Success *bool           `json:"success"`
		Result  json.RawMessage `json:"result"`
		Message string          `json:"message"`
	}
	if json.Unmarshal(data, &env) != nil || env.Success == nil || !isJSONObject(env.Result) {
		return data
	}
	result := bytes.TrimSpace(env.Result)
	msg := strings.TrimSpace(env.Message)
	if msg == "" {
		return result
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(result, &fields) != nil {
		return result
	}
	if _, has := fields["message"]; has {
		return result
	}
	encoded, _ := json.Marshal(msg)
	out := make([]byte, 0, len(result)+len(encoded)+12)
	out = append(out, result[:len(result)-1]...) // drop the closing brace
	if len(fields) > 0 {
		out = append(out, ',')
	}
	out = append(out, `"message":`...)
	out = append(out, encoded...)
	out = append(out, '}')
	return out
}
