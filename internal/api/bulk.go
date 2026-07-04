package api

import (
	"encoding/json"
	"net/url"
	"strings"
)

// BulkLookup streams enrichment results for up to 1,000 emails, invoking
// onPerson for each result as it arrives over the event stream.
func (c *Client) BulkLookup(emails, fields []string, onPerson func(json.RawMessage) error) error {
	path := "/api/agent/bulk-lookup"
	if len(fields) > 0 {
		q := url.Values{}
		q.Set("fields", strings.Join(fields, ","))
		path += "?" + q.Encode()
	}
	return c.stream(path, map[string][]string{"emails": emails}, func(eventType string, event json.RawMessage) error {
		// The stream emits start/result/error/done events. Only "result"
		// events carry an enriched person, nested under the "data" field.
		if eventType != "result" {
			return nil
		}
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if json.Unmarshal(event, &envelope) == nil && envelope.Data != nil {
			return onPerson(envelope.Data)
		}
		return nil
	})
}

// BulkSearch aggregates a bulk search stream (google/company/domain/ip) into a
// single result set.
func (c *Client) BulkSearch(path string, queries []string) (json.RawMessage, error) {
	var results []json.RawMessage
	err := c.stream(path, map[string][]string{"queries": queries}, func(eventType string, event json.RawMessage) error {
		// Only "result" events carry search results; skip start/done.
		if eventType != "result" {
			return nil
		}
		results = append(results, event)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(results)
}

func (c *Client) BulkGoogleSearch(queries []string) (json.RawMessage, error) {
	return c.BulkSearch("/api/bulk-google-search", queries)
}

func (c *Client) BulkCompanySearch(queries []string) (json.RawMessage, error) {
	return c.BulkSearch("/api/bulk-company-search", queries)
}

func (c *Client) BulkDomainSearch(queries []string) (json.RawMessage, error) {
	return c.BulkSearch("/api/bulk-domain-search", queries)
}

func (c *Client) BulkIPSearch(queries []string) (json.RawMessage, error) {
	return c.BulkSearch("/api/bulk-ip-search", queries)
}
