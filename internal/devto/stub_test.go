package devto

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// seen is one request captured by the stub dev.to server.
type seen struct {
	Method string
	Path   string // escaped path, as sent on the wire
	Query  string
	Header http.Header
	Body   []byte
}

// stub is an in-process dev.to: it records every request and answers with a
// fixed status and body. No network, no API key.
type stub struct {
	status int
	body   string
	reqs   []seen
}

func (s *stub) last(t *testing.T) seen {
	t.Helper()
	if len(s.reqs) == 0 {
		t.Fatal("no request reached the stub")
	}
	return s.reqs[len(s.reqs)-1]
}

// newStubClient returns a Client whose HTTP calls land on the stub.
func newStubClient(t *testing.T, apiKey string, status int, body string) (*Client, *stub) {
	t.Helper()
	st := &stub{status: status, body: body}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		st.reqs = append(st.reqs, seen{
			Method: r.Method,
			Path:   r.URL.EscapedPath(),
			Query:  r.URL.RawQuery,
			Header: r.Header.Clone(),
			Body:   b,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(st.status)
		_, _ = w.Write([]byte(st.body))
	}))
	t.Cleanup(srv.Close)

	c := New(apiKey)
	c.http.Transport = toHost{strings.TrimPrefix(srv.URL, "http://")}
	return c, st
}

// toolSession registers the real tools on an MCP server backed by the stub and
// returns a connected in-memory client session.
func toolSession(t *testing.T, apiKey string, status int, body string) (*mcp.ClientSession, *stub) {
	t.Helper()
	c, st := newStubClient(t, apiKey, status, body)

	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	Register(s, c)

	ctx := context.Background()
	srvT, cliT := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, srvT, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "0"}, nil).Connect(ctx, cliT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs, st
}

// callTool invokes a tool and returns (isError, text).
func callTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) (bool, string) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return true, err.Error()
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return res.IsError, b.String()
}

func decodeArticle(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var payload struct {
		Article map[string]any `json:"article"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("body is not {\"article\":{...}}: %v (%s)", err, body)
	}
	if payload.Article == nil {
		t.Fatalf("missing article key in body: %s", body)
	}
	return payload.Article
}
