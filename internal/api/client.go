package api

import (
	"net/http"
	"strings"
	"time"
)

// Version is stamped into the User-Agent header. Overridden by the cmd package.
var Version = "dev"

const (
	streamTimeout  = 5 * time.Minute
	maxRetries     = 3
	initialBackoff = 500 * time.Millisecond
	maxBackoff     = 10 * time.Second
)

// RequestTimeout is the per-request HTTP timeout. Defaults to 90s and can be
// overridden (e.g. by the CLI's --timeout flag) before a client is created.
var RequestTimeout = 90 * time.Second

var retryableStatus = map[int]bool{
	http.StatusTooManyRequests:     true,
	http.StatusInternalServerError: true,
	http.StatusBadGateway:          true,
	http.StatusServiceUnavailable:  true,
	http.StatusGatewayTimeout:      true,
}

type Client struct {
	BaseURL      string
	APIKey       string
	UserAgent    string
	HTTPClient   *http.Client
	streamClient *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL:      strings.TrimRight(baseURL, "/"),
		APIKey:       apiKey,
		UserAgent:    "encrata-cli/" + Version,
		HTTPClient:   &http.Client{Timeout: RequestTimeout},
		streamClient: &http.Client{Timeout: streamTimeout},
	}
}
