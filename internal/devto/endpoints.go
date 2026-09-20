package devto

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// This file maps each dev.to endpoint to one small method. They only build the
// path and query, then delegate to get/writeArticle. Response bodies flow back
// as raw JSON by design (see package doc): the caller is an LLM that reads JSON.

// --- reads (public) ---

func (c *Client) GetArticles(ctx context.Context, params url.Values) (json.RawMessage, error) {
	return c.get(ctx, "articles", params)
}

func (c *Client) GetArticleByID(ctx context.Context, id int) (json.RawMessage, error) {
	return c.get(ctx, "articles/"+strconv.Itoa(id), nil)
}

func (c *Client) GetArticleByPath(ctx context.Context, path string) (json.RawMessage, error) {
	return c.get(ctx, "articles/"+url.PathEscape(path), nil)
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
	return c.get(ctx, "search/feed_content", params)
}

// --- writes (require API key) ---

func (c *Client) CreateArticle(ctx context.Context, article map[string]any) (json.RawMessage, error) {
	return c.writeArticle(ctx, "POST", "articles", article)
}

func (c *Client) UpdateArticle(ctx context.Context, id int, article map[string]any) (json.RawMessage, error) {
	return c.writeArticle(ctx, "PUT", "articles/"+strconv.Itoa(id), article)
}
