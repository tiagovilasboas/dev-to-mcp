// Package devto is a thin HTTP client for the dev.to (Forem) API. It knows
// nothing about MCP: it builds requests, sends them, and returns the raw JSON
// body. Read endpoints are public; write endpoints require the API key.
package devto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseURL     = "https://dev.to/api/"
	apiV1Accept = "application/vnd.forem.api-v1+json"
)

// Client talks to the dev.to API. The zero value is not usable; use New.
type Client struct {
	http   *http.Client
	apiKey string // empty means read-only (write calls will fail fast)
}

// New builds a Client. apiKey may be empty for read-only usage.
func New(apiKey string) *Client {
	return &Client{
		http:   &http.Client{Timeout: 20 * time.Second},
		apiKey: apiKey,
	}
}

// get performs a public GET and returns the raw JSON body. This is the single
// place read endpoints share, so query handling and error mapping live once.
func (c *Client) get(ctx context.Context, path string, params url.Values) (json.RawMessage, error) {
	endpoint := baseURL + path
	if q := params.Encode(); q != "" {
		endpoint += "?" + q
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// writeArticle is the single path for POST/PUT of an article. The body is
// always {"article": {...}} and both create and update reuse it.
func (c *Client) writeArticle(ctx context.Context, method, path string, article map[string]any) (json.RawMessage, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("no API key configured: store it in the macOS Keychain (service=dev-to-mcp) or set DEV_TO_API_KEY")
	}

	payload, err := json.Marshal(map[string]any{"article": article})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("Accept", apiV1Accept)
	return c.do(req)
}

// do sends the request and maps non-2xx responses to errors carrying the
// body, so callers surface dev.to's own message (e.g. validation errors).
func (c *Client) do(req *http.Request) (json.RawMessage, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dev.to %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}
