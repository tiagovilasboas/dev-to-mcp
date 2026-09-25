package devto

import "net/url"

// Input types are the tool contracts. The `jsonschema` struct tags become the
// schema the MCP client shows the model, so descriptions live here.
// Each type owns its own query()/article() builder to keep tools.go declarative.

type GetArticlesInput struct {
	Username string `json:"username,omitempty" jsonschema:"filter by author username"`
	Tag      string `json:"tag,omitempty" jsonschema:"filter by a single tag"`
	State    string `json:"state,omitempty" jsonschema:"article state: fresh, rising, or all"`
	Top      int    `json:"top,omitempty" jsonschema:"top articles over the last N days"`
	Page     int    `json:"page,omitempty" jsonschema:"pagination page (default 1)"`
	PerPage  int    `json:"per_page,omitempty" jsonschema:"articles per page (default 30, max 1000)"`
}

func (in GetArticlesInput) query() url.Values {
	v := url.Values{}
	setStr(v, "username", in.Username)
	setStr(v, "tag", in.Tag)
	setStr(v, "state", in.State)
	setInt(v, "top", in.Top)
	setInt(v, "page", in.Page)
	setInt(v, "per_page", in.PerPage)
	return v
}

type GetArticleInput struct {
	ID   int    `json:"id,omitempty" jsonschema:"article numeric id"`
	Path string `json:"path,omitempty" jsonschema:"article path, e.g. username/article-slug"`
}

type GetUserInput struct {
	ID       int    `json:"id,omitempty" jsonschema:"user numeric id"`
	Username string `json:"username,omitempty" jsonschema:"dev.to username"`
}

type PageInput struct {
	Page    int `json:"page,omitempty" jsonschema:"pagination page (default 1)"`
	PerPage int `json:"per_page,omitempty" jsonschema:"items per page"`
}

func (in PageInput) query() url.Values {
	v := url.Values{}
	setInt(v, "page", in.Page)
	setInt(v, "per_page", in.PerPage)
	return v
}

type GetCommentsInput struct {
	ArticleID int `json:"article_id" jsonschema:"article numeric id to fetch comments for"`
}

type SearchInput struct {
	Query   string `json:"q" jsonschema:"search query (required); matches title, tags and body"`
	Top     int    `json:"top,omitempty" jsonschema:"only articles published in the last N days"`
	Page    int    `json:"page,omitempty" jsonschema:"pagination page (default 1)"`
	PerPage int    `json:"per_page,omitempty" jsonschema:"results per page (default 30)"`
}

func (in SearchInput) query() url.Values {
	v := url.Values{}
	setStr(v, "q", in.Query)
	setInt(v, "top", in.Top)
	setInt(v, "page", in.Page)
	setInt(v, "per_page", in.PerPage)
	return v
}

// CreateArticleInput and UpdateArticleInput share the article body builder via
// articleFields. Only presence differs: create needs title+body, update patches.

type CreateArticleInput struct {
	Title        string   `json:"title" jsonschema:"article title"`
	BodyMarkdown string   `json:"body_markdown" jsonschema:"article body in Markdown"`
	Published    bool     `json:"published,omitempty" jsonschema:"publish immediately (default false = draft)"`
	Tags         []string `json:"tags,omitempty" jsonschema:"up to 4 tags"`
	Series       string   `json:"series,omitempty" jsonschema:"series name to group the article"`
	CanonicalURL string   `json:"canonical_url,omitempty" jsonschema:"canonical URL if cross-posting"`
	Description  string   `json:"description,omitempty" jsonschema:"short description / subtitle"`
}

func (in CreateArticleInput) article() map[string]any {
	a := articleFields(in.Tags, in.Series, in.CanonicalURL, in.Description)
	a["title"] = in.Title
	a["body_markdown"] = in.BodyMarkdown
	a["published"] = in.Published
	return a
}

type UpdateArticleInput struct {
	ID           int      `json:"id" jsonschema:"article numeric id to update"`
	Title        string   `json:"title,omitempty" jsonschema:"new title"`
	BodyMarkdown string   `json:"body_markdown,omitempty" jsonschema:"new body in Markdown"`
	Published    *bool    `json:"published,omitempty" jsonschema:"set true to publish a draft, false to unpublish"`
	Tags         []string `json:"tags,omitempty" jsonschema:"new list of tags (max 4)"`
	Series       string   `json:"series,omitempty" jsonschema:"series name"`
	CanonicalURL string   `json:"canonical_url,omitempty" jsonschema:"canonical URL"`
	Description  string   `json:"description,omitempty" jsonschema:"short description"`
}

func (in UpdateArticleInput) article() map[string]any {
	a := articleFields(in.Tags, in.Series, in.CanonicalURL, in.Description)
	setField(a, "title", in.Title)
	setField(a, "body_markdown", in.BodyMarkdown)
	if in.Published != nil {
		a["published"] = *in.Published
	}
	return a
}

// articleFields collects the fields create and update have in common, only
// including the ones actually provided (dev.to treats present keys as updates).
func articleFields(tags []string, series, canonicalURL, description string) map[string]any {
	a := map[string]any{}
	if len(tags) > 0 {
		a["tags"] = tags
	}
	setField(a, "series", series)
	setField(a, "canonical_url", canonicalURL)
	setField(a, "description", description)
	return a
}

func setField(a map[string]any, key, val string) {
	if val != "" {
		a[key] = val
	}
}
