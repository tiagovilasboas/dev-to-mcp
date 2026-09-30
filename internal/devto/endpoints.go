package devto

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// This file maps each dev.to endpoint to one small method. They only build the
// path and query, then delegate to get/writeArticle. Response bodies flow back
// as raw JSON by design (see package doc): the caller is an LLM that reads JSON.

// --- reads (public) ---

func (c *Client) GetArticles(ctx context.Context, params url.Values) (json.RawMessage, error) {
	return c.get(ctx, "articles", params)
}

// GetMyArticles returns the authenticated user's articles. State must be one
// of all, published, or unpublished; unpublished is useful for finding drafts.
func (c *Client) GetMyArticles(ctx context.Context, state string, params url.Values) (json.RawMessage, error) {
	if state != "all" && state != "published" && state != "unpublished" {
		return nil, fmt.Errorf("state must be all, published, or unpublished")
	}
	return c.getAuthenticated(ctx, "articles/me/"+state, params)
}

func (c *Client) GetArticleByID(ctx context.Context, id int) (json.RawMessage, error) {
	return c.get(ctx, "articles/"+strconv.Itoa(id), nil)
}

func (c *Client) GetArticleByPath(ctx context.Context, path string) (json.RawMessage, error) {
	// path is "username/article-slug": escape each segment individually so the
	// separating slash is preserved and the DEV.to API can route correctly.
	// A leading slash (the form DEV.to itself returns in "path") is dropped.
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
	escaped := url.PathEscape(parts[0])
	if len(parts) == 2 {
		escaped += "/" + url.PathEscape(parts[1])
	}
	return c.get(ctx, "articles/"+escaped, nil)
}

func (c *Client) GetUserByID(ctx context.Context, id int) (json.RawMessage, error) {
	return c.get(ctx, "users/"+strconv.Itoa(id), nil)
}

func (c *Client) GetUserByUsername(ctx context.Context, username string) (json.RawMessage, error) {
	params := url.Values{"url": {username}}
	return c.get(ctx, "users/by_username", params)
}

func (c *Client) GetTags(ctx context.Context, params url.Values) (json.RawMessage, error) {
	return c.get(ctx, "tags", params)
}

func (c *Client) GetComments(ctx context.Context, articleID int) (json.RawMessage, error) {
	params := url.Values{"a_id": {strconv.Itoa(articleID)}}
	return c.get(ctx, "comments", params)
}

func (c *Client) SearchArticles(ctx context.Context, params url.Values) (json.RawMessage, error) {
	return c.get(ctx, "articles/search", params)
}

// --- writes (require API key) ---

func (c *Client) CreateArticle(ctx context.Context, article map[string]any) (json.RawMessage, error) {
	return c.writeArticle(ctx, "POST", "articles", article)
}

func (c *Client) UpdateArticle(ctx context.Context, id int, article map[string]any) (json.RawMessage, error) {
	return c.writeArticle(ctx, "PUT", "articles/"+strconv.Itoa(id), article)
}
