package devto

import (
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestRegisterExposesNineTools(t *testing.T) {
	cs, _ := toolSession(t, "", http.StatusOK, `{}`)
	res, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
		if tool.Description == "" || tool.InputSchema == nil {
			t.Errorf("%s: missing description or input schema", tool.Name)
		}
	}
	sort.Strings(names)
	want := []string{"create_article", "get_article", "get_articles", "get_comments", "get_my_articles", "get_tags", "get_user", "search_articles", "update_article"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("tools = %v, want %v", names, want)
	}
}

// Every read tool: arguments become the right dev.to path/query and the body
// comes back verbatim as text content.
func TestReadToolsHitTheRightEndpoint(t *testing.T) {
	const body = `[{"id":1}]`
	cases := []struct {
		tool      string
		args      map[string]any
		wantPath  string
		wantQuery url.Values
	}{
		{"get_articles", map[string]any{"username": "ana", "tag": "go", "state": "rising", "top": 7, "page": 2, "per_page": 10},
			"/api/articles", url.Values{"username": {"ana"}, "tag": {"go"}, "state": {"rising"}, "top": {"7"}, "page": {"2"}, "per_page": {"10"}}},
		{"get_articles", map[string]any{}, "/api/articles", url.Values{}},
		{"get_article", map[string]any{"id": 42}, "/api/articles/42", url.Values{}},
		{"get_article", map[string]any{"path": "ana/hello-1a2b"}, "/api/articles/ana/hello-1a2b", url.Values{}},
		{"get_user", map[string]any{"id": 9}, "/api/users/9", url.Values{}},
		{"get_user", map[string]any{"username": "ana"}, "/api/users/by_username", url.Values{"url": {"ana"}}},
		{"get_tags", map[string]any{"page": 3, "per_page": 20}, "/api/tags", url.Values{"page": {"3"}, "per_page": {"20"}}},
		{"get_comments", map[string]any{"article_id": 11}, "/api/comments", url.Values{"a_id": {"11"}}},
		{"search_articles", map[string]any{"q": "mcp"}, "/api/articles/search", url.Values{"q": {"mcp"}}},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			cs, st := toolSession(t, "", http.StatusOK, body)
			isErr, text := callTool(t, cs, tc.tool, tc.args)
			if isErr {
				t.Fatalf("tool error: %s", text)
			}
			if text != body {
				t.Errorf("text = %s, want %s", text, body)
			}
			r := st.last(t)
			if r.Method != http.MethodGet || r.Path != tc.wantPath {
				t.Errorf("got %s %s, want GET %s", r.Method, r.Path, tc.wantPath)
			}
			if q, _ := url.ParseQuery(r.Query); q.Encode() != tc.wantQuery.Encode() {
				t.Errorf("query = %q, want %q", q.Encode(), tc.wantQuery.Encode())
			}
			if r.Header.Get("api-key") != "" {
				t.Error("read tools must not send the api key")
			}
		})
	}
}

func TestGetMyArticlesTool(t *testing.T) {
	t.Run("defaults to unpublished and authenticates", func(t *testing.T) {
		cs, st := toolSession(t, "secret", http.StatusOK, `[]`)
		isErr, text := callTool(t, cs, "get_my_articles", map[string]any{"page": 2, "per_page": 5})
		if isErr {
			t.Fatalf("tool error: %s", text)
		}
		r := st.last(t)
		if r.Path != "/api/articles/me/unpublished" {
			t.Errorf("path = %s", r.Path)
		}
		if q, _ := url.ParseQuery(r.Query); q.Encode() != "page=2&per_page=5" {
			t.Errorf("query = %s", r.Query)
		}
		if r.Header.Get("api-key") != "secret" {
			t.Error("api-key header missing")
		}
	})

	t.Run("honours explicit state", func(t *testing.T) {
		cs, st := toolSession(t, "secret", http.StatusOK, `[]`)
		if isErr, text := callTool(t, cs, "get_my_articles", map[string]any{"state": "published"}); isErr {
			t.Fatalf("tool error: %s", text)
		}
		if r := st.last(t); r.Path != "/api/articles/me/published" {
			t.Errorf("path = %s", r.Path)
		}
	})

	t.Run("invalid state is a tool error", func(t *testing.T) {
		cs, st := toolSession(t, "secret", http.StatusOK, `[]`)
		if isErr, _ := callTool(t, cs, "get_my_articles", map[string]any{"state": "drafts"}); !isErr {
			t.Error("want tool error")
		}
		if len(st.reqs) != 0 {
			t.Error("no request expected")
		}
	})

	t.Run("no api key is a tool error, not a crash", func(t *testing.T) {
		cs, st := toolSession(t, "", http.StatusOK, `[]`)
		isErr, text := callTool(t, cs, "get_my_articles", nil)
		if !isErr || !strings.Contains(text, "no API key") {
			t.Errorf("isErr=%v text=%q", isErr, text)
		}
		if len(st.reqs) != 0 {
			t.Error("no request expected")
		}
	})
}

func TestToolValidationErrorsSkipHTTP(t *testing.T) {
	cases := []struct {
		tool string
		args map[string]any
		want string
	}{
		{"get_article", map[string]any{}, "provide either id or path"},
		{"get_user", map[string]any{}, "provide either id or username"},
		{"get_comments", map[string]any{"article_id": 0}, "article_id must be a positive integer"},
		{"update_article", map[string]any{"id": 0, "title": "x"}, "id must be a positive integer"},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			cs, st := toolSession(t, "secret", http.StatusOK, `{}`)
			isErr, text := callTool(t, cs, tc.tool, tc.args)
			if !isErr || !strings.Contains(text, tc.want) {
				t.Errorf("isErr=%v text=%q, want error containing %q", isErr, text, tc.want)
			}
			if len(st.reqs) != 0 {
				t.Errorf("no request expected, got %d", len(st.reqs))
			}
		})
	}
}

func TestUpstreamErrorSurfacesAsToolError(t *testing.T) {
	cs, _ := toolSession(t, "", http.StatusNotFound, `{"error":"not found","status":404}`)
	isErr, text := callTool(t, cs, "get_article", map[string]any{"id": 1})
	if !isErr || !strings.Contains(text, "404") || !strings.Contains(text, "not found") {
		t.Errorf("isErr=%v text=%q", isErr, text)
	}
}

func TestCreateArticleTool(t *testing.T) {
	t.Run("full payload", func(t *testing.T) {
		cs, st := toolSession(t, "secret", http.StatusCreated, `{"id":99}`)
		isErr, text := callTool(t, cs, "create_article", map[string]any{
			"title": "Hello", "body_markdown": "# Hi", "published": true,
			"tags": []string{"go", "mcp"}, "series": "S", "canonical_url": "https://x.dev/a", "description": "D",
		})
		if isErr || text != `{"id":99}` {
			t.Fatalf("isErr=%v text=%s", isErr, text)
		}
		r := st.last(t)
		if r.Method != http.MethodPost || r.Path != "/api/articles" || r.Header.Get("api-key") != "secret" {
			t.Errorf("got %s %s key=%q", r.Method, r.Path, r.Header.Get("api-key"))
		}
		want := map[string]any{
			"title": "Hello", "body_markdown": "# Hi", "published": true,
			"tags": []any{"go", "mcp"}, "series": "S", "canonical_url": "https://x.dev/a", "description": "D",
		}
		if a := decodeArticle(t, r.Body); !reflect.DeepEqual(a, want) {
			t.Errorf("article = %v\nwant    %v", a, want)
		}
	})

	t.Run("draft by default, optional fields omitted", func(t *testing.T) {
		cs, st := toolSession(t, "secret", http.StatusCreated, `{}`)
		if isErr, text := callTool(t, cs, "create_article", map[string]any{"title": "T", "body_markdown": "B"}); isErr {
			t.Fatalf("tool error: %s", text)
		}
		want := map[string]any{"title": "T", "body_markdown": "B", "published": false}
		if a := decodeArticle(t, st.last(t).Body); !reflect.DeepEqual(a, want) {
			t.Errorf("article = %v, want %v", a, want)
		}
	})

	t.Run("no api key is a tool error", func(t *testing.T) {
		cs, st := toolSession(t, "", http.StatusCreated, `{}`)
		if isErr, _ := callTool(t, cs, "create_article", map[string]any{"title": "T", "body_markdown": "B"}); !isErr {
			t.Error("want tool error")
		}
		if len(st.reqs) != 0 {
			t.Error("no request expected")
		}
	})
}

func TestUpdateArticleTool(t *testing.T) {
	t.Run("only provided fields are sent", func(t *testing.T) {
		cs, st := toolSession(t, "secret", http.StatusOK, `{"id":5}`)
		if isErr, text := callTool(t, cs, "update_article", map[string]any{"id": 5, "title": "New"}); isErr {
			t.Fatalf("tool error: %s", text)
		}
		r := st.last(t)
		if r.Method != http.MethodPut || r.Path != "/api/articles/5" {
			t.Errorf("got %s %s", r.Method, r.Path)
		}
		if a := decodeArticle(t, r.Body); !reflect.DeepEqual(a, map[string]any{"title": "New"}) {
			t.Errorf("article = %v", a)
		}
	})

	t.Run("published false is sent explicitly", func(t *testing.T) {
		cs, st := toolSession(t, "secret", http.StatusOK, `{}`)
		if isErr, text := callTool(t, cs, "update_article", map[string]any{
			"id": 5, "published": false, "body_markdown": "B", "tags": []string{"a"}, "series": "S", "canonical_url": "C", "description": "D",
		}); isErr {
			t.Fatalf("tool error: %s", text)
		}
		want := map[string]any{"published": false, "body_markdown": "B", "tags": []any{"a"}, "series": "S", "canonical_url": "C", "description": "D"}
		if a := decodeArticle(t, st.last(t).Body); !reflect.DeepEqual(a, want) {
			t.Errorf("article = %v, want %v", a, want)
		}
	})

	t.Run("published true publishes a draft", func(t *testing.T) {
		cs, st := toolSession(t, "secret", http.StatusOK, `{}`)
		if isErr, text := callTool(t, cs, "update_article", map[string]any{"id": 5, "published": true}); isErr {
			t.Fatalf("tool error: %s", text)
		}
		if a := decodeArticle(t, st.last(t).Body); !reflect.DeepEqual(a, map[string]any{"published": true}) {
			t.Errorf("article = %v", a)
		}
	})
}
