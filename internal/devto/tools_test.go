package devto

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toHost sends every request to the stub server, keeping path and query.
type toHost struct{ host string }

func (t toHost) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.URL.Scheme, r.URL.Host = "http", t.host
	return http.DefaultTransport.RoundTrip(r)
}

// session registers the real tools against a client whose HTTP calls hit a
// stub, and returns an in-memory MCP session plus the last request URL seen.
func session(t *testing.T, body string) (*mcp.ClientSession, *url.URL) {
	t.Helper()
	got := &url.URL{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*got = *r.URL
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	c := New("")
	c.http.Transport = toHost{strings.TrimPrefix(srv.URL, "http://")}

	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	Register(s, c)

	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs, got
}

func call(t *testing.T, cs *mcp.ClientSession, args map[string]any) (*mcp.CallToolResult, string) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "search_articles", Arguments: args})
	if err != nil {
		return nil, err.Error()
	}
	return res, res.Content[0].(*mcp.TextContent).Text
}

// dev.to serves full-text search at /api/articles/search; the old
// /api/search/feed_content path is a 404.
func TestSearchArticlesHitsArticlesSearch(t *testing.T) {
	const body = `[{"type_of":"article","id":1,"title":"Go generics","path":"/u/go-generics"}]`
	cs, got := session(t, body)

	res, text := call(t, cs, map[string]any{"q": "go generics", "top": 30, "page": 2, "per_page": 5})
	if res == nil || res.IsError {
		t.Fatalf("tool error: %s", text)
	}
	if got.Path != "/api/articles/search" {
		t.Errorf("path = %q, want /api/articles/search", got.Path)
	}
	want := url.Values{"q": {"go generics"}, "top": {"30"}, "page": {"2"}, "per_page": {"5"}}
	if q := got.Query(); q.Encode() != want.Encode() {
		t.Errorf("query = %q, want %q", q.Encode(), want.Encode())
	}
	if text != body {
		t.Errorf("body not passed through verbatim:\n got %s\nwant %s", text, body)
	}
}

func TestSearchArticlesRejectsMissingQueryAndUnknownFilters(t *testing.T) {
	cs, got := session(t, `[]`)

	for _, args := range []map[string]any{
		{"top": 7},                            // q missing: schema rejects
		{"q": ""},                             // q empty: handler rejects
		{"q": "go", "search_fields": "title"}, // not a dev.to search param
	} {
		res, text := call(t, cs, args)
		if res == nil || !res.IsError {
			t.Errorf("%v: want tool error, got %q", args, text)
		}
	}
	if got.Path != "" {
		t.Errorf("no request expected, got %q", got.Path)
	}
}
