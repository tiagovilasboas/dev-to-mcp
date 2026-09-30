package devto

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestNewDefaults(t *testing.T) {
	c := New("k")
	if c.apiKey != "k" {
		t.Errorf("apiKey = %q", c.apiKey)
	}
	if c.http == nil || c.http.Timeout <= 0 {
		t.Errorf("http client must have a timeout, got %+v", c.http)
	}
}

func TestGetPublicSendsNoAPIKey(t *testing.T) {
	c, st := newStubClient(t, "secret", http.StatusOK, `[]`)

	raw, err := c.GetArticles(context.Background(), url.Values{"tag": {"go"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `[]` {
		t.Errorf("raw = %s", raw)
	}
	r := st.last(t)
	if r.Method != http.MethodGet || r.Path != "/api/articles" || r.Query != "tag=go" {
		t.Errorf("got %s %s?%s", r.Method, r.Path, r.Query)
	}
	if r.Header.Get("api-key") != "" {
		t.Error("public read must not leak the api-key header")
	}
}

func TestGetWithoutParamsHasNoQuery(t *testing.T) {
	c, st := newStubClient(t, "", http.StatusOK, `{}`)
	if _, err := c.GetArticleByID(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	if r := st.last(t); r.Path != "/api/articles/42" || r.Query != "" {
		t.Errorf("got %s?%s", r.Path, r.Query)
	}
}

func TestNon2xxBecomesErrorWithBody(t *testing.T) {
	c, _ := newStubClient(t, "", http.StatusUnprocessableEntity, "  {\"error\":\"title can't be blank\"}\n")

	_, err := c.GetTags(context.Background(), nil)
	if err == nil {
		t.Fatal("want error on 422")
	}
	msg := err.Error()
	if !strings.Contains(msg, "422") || !strings.HasSuffix(msg, `{"error":"title can't be blank"}`) {
		t.Errorf("error = %q, want status and trimmed body", msg)
	}
}

func TestTransportErrorIsReturned(t *testing.T) {
	c := New("")
	c.http.Transport = failRT{}
	if _, err := c.GetTags(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %v, want transport error", err)
	}
	c = New("k")
	c.http.Transport = failRT{}
	if _, err := c.GetMyArticles(context.Background(), "all", nil); err == nil {
		t.Error("authenticated get: want transport error")
	}
	if _, err := c.CreateArticle(context.Background(), map[string]any{}); err == nil {
		t.Error("write: want transport error")
	}
}

func TestBadContextFailsBeforeSending(t *testing.T) {
	c, st := newStubClient(t, "k", http.StatusOK, `{}`)
	// A nil context makes NewRequestWithContext fail before anything is sent.
	var nilCtx context.Context
	if _, err := c.get(nilCtx, "tags", nil); err == nil {
		t.Error("get: want error for nil context")
	}
	if _, err := c.getAuthenticated(nilCtx, "articles/me/all", nil); err == nil {
		t.Error("getAuthenticated: want error for nil context")
	}
	if _, err := c.writeArticle(nilCtx, http.MethodPost, "articles", map[string]any{}); err == nil {
		t.Error("writeArticle: want error for nil context")
	}
	if len(st.reqs) != 0 {
		t.Errorf("no request expected, got %d", len(st.reqs))
	}
}

func TestWriteRejectsUnmarshalableArticle(t *testing.T) {
	c, st := newStubClient(t, "k", http.StatusOK, `{}`)
	if _, err := c.CreateArticle(context.Background(), map[string]any{"bad": make(chan int)}); err == nil {
		t.Error("want marshal error")
	}
	if len(st.reqs) != 0 {
		t.Error("nothing should be sent when the body cannot be encoded")
	}
}

func TestReadBodyErrorIsReturned(t *testing.T) {
	c := New("")
	c.http.Transport = badBodyRT{}
	if _, err := c.GetTags(context.Background(), nil); err == nil {
		t.Error("want body read error")
	}
}

func TestEndpointsBuildPathsAndQueries(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		call      func(*Client) error
		wantPath  string
		wantQuery string
	}{
		{"articles", func(c *Client) error { _, err := c.GetArticles(ctx, url.Values{"page": {"2"}}); return err }, "/api/articles", "page=2"},
		{"article by id", func(c *Client) error { _, err := c.GetArticleByID(ctx, 7); return err }, "/api/articles/7", ""},
		{"article by path", func(c *Client) error { _, err := c.GetArticleByPath(ctx, "ana/hello-world-1a2b"); return err }, "/api/articles/ana/hello-world-1a2b", ""},
		{"article by path escapes segments", func(c *Client) error { _, err := c.GetArticleByPath(ctx, "a b/c?d"); return err }, "/api/articles/a%20b/c%3Fd", ""},
		// DEV.to returns "path":"/user/slug"; passing it back must not produce
		// "articles//user%2Fslug", which the API answers with 404.
		{"article by path with leading slash", func(c *Client) error { _, err := c.GetArticleByPath(ctx, "/ana/hello-world-1a2b"); return err }, "/api/articles/ana/hello-world-1a2b", ""},
		{"article by path without slash", func(c *Client) error { _, err := c.GetArticleByPath(ctx, "slug"); return err }, "/api/articles/slug", ""},
		{"user by id", func(c *Client) error { _, err := c.GetUserByID(ctx, 9); return err }, "/api/users/9", ""},
		{"user by username", func(c *Client) error { _, err := c.GetUserByUsername(ctx, "ana"); return err }, "/api/users/by_username", "url=ana"},
		{"tags", func(c *Client) error { _, err := c.GetTags(ctx, url.Values{"per_page": {"3"}}); return err }, "/api/tags", "per_page=3"},
		{"comments", func(c *Client) error { _, err := c.GetComments(ctx, 11); return err }, "/api/comments", "a_id=11"},
		{"search", func(c *Client) error { _, err := c.SearchArticles(ctx, url.Values{"q": {"go"}}); return err }, "/api/articles/search", "q=go"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, st := newStubClient(t, "", http.StatusOK, `{}`)
			if err := tc.call(c); err != nil {
				t.Fatal(err)
			}
			r := st.last(t)
			if r.Method != http.MethodGet || r.Path != tc.wantPath || r.Query != tc.wantQuery {
				t.Errorf("got %s %s?%s, want GET %s?%s", r.Method, r.Path, r.Query, tc.wantPath, tc.wantQuery)
			}
		})
	}
}

func TestGetMyArticles(t *testing.T) {
	ctx := context.Background()

	t.Run("sends api key and forem accept", func(t *testing.T) {
		for _, state := range []string{"all", "published", "unpublished"} {
			c, st := newStubClient(t, "secret", http.StatusOK, `[]`)
			if _, err := c.GetMyArticles(ctx, state, url.Values{"page": {"1"}}); err != nil {
				t.Fatal(err)
			}
			r := st.last(t)
			if r.Path != "/api/articles/me/"+state || r.Query != "page=1" {
				t.Errorf("got %s?%s", r.Path, r.Query)
			}
			if r.Header.Get("api-key") != "secret" || r.Header.Get("Accept") != apiV1Accept {
				t.Errorf("headers = %v", r.Header)
			}
		}
	})

	t.Run("rejects unknown state without a request", func(t *testing.T) {
		c, st := newStubClient(t, "secret", http.StatusOK, `[]`)
		if _, err := c.GetMyArticles(ctx, "drafts", nil); err == nil {
			t.Error("want error for invalid state")
		}
		if len(st.reqs) != 0 {
			t.Error("no request expected")
		}
	})

	t.Run("fails fast without api key", func(t *testing.T) {
		c, st := newStubClient(t, "", http.StatusOK, `[]`)
		_, err := c.GetMyArticles(ctx, "all", nil)
		if err == nil || !strings.Contains(err.Error(), "no API key") {
			t.Errorf("err = %v", err)
		}
		if len(st.reqs) != 0 {
			t.Error("no request expected")
		}
	})
}

func TestWrites(t *testing.T) {
	ctx := context.Background()

	t.Run("create posts article envelope with auth headers", func(t *testing.T) {
		c, st := newStubClient(t, "secret", http.StatusCreated, `{"id":1}`)
		raw, err := c.CreateArticle(ctx, map[string]any{"title": "T"})
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) != `{"id":1}` {
			t.Errorf("raw = %s", raw)
		}
		r := st.last(t)
		if r.Method != http.MethodPost || r.Path != "/api/articles" {
			t.Errorf("got %s %s", r.Method, r.Path)
		}
		for k, want := range map[string]string{"api-key": "secret", "Accept": apiV1Accept, "Content-Type": "application/json"} {
			if got := r.Header.Get(k); got != want {
				t.Errorf("header %s = %q, want %q", k, got, want)
			}
		}
		if a := decodeArticle(t, r.Body); a["title"] != "T" {
			t.Errorf("article = %v", a)
		}
	})

	t.Run("update puts to article id", func(t *testing.T) {
		c, st := newStubClient(t, "secret", http.StatusOK, `{"id":5}`)
		if _, err := c.UpdateArticle(ctx, 5, map[string]any{"published": true}); err != nil {
			t.Fatal(err)
		}
		r := st.last(t)
		if r.Method != http.MethodPut || r.Path != "/api/articles/5" {
			t.Errorf("got %s %s", r.Method, r.Path)
		}
		if a := decodeArticle(t, r.Body); a["published"] != true {
			t.Errorf("article = %v", a)
		}
	})

	t.Run("fails fast without api key", func(t *testing.T) {
		c, st := newStubClient(t, "", http.StatusOK, `{}`)
		if _, err := c.UpdateArticle(ctx, 5, map[string]any{}); err == nil || !strings.Contains(err.Error(), "no API key") {
			t.Errorf("err = %v", err)
		}
		if len(st.reqs) != 0 {
			t.Error("no request expected")
		}
	})
}

type failRT struct{}

func (failRT) RoundTrip(*http.Request) (*http.Response, error) { return nil, errors.New("boom") }

type badBodyRT struct{}

func (badBodyRT) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: errBody{}, Header: http.Header{}, Request: r}, nil
}

type errBody struct{}

func (errBody) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errBody) Close() error             { return nil }
