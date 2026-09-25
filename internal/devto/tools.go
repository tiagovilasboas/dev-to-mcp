package devto

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register wires every dev.to tool onto the MCP server. Input structs are
// typed (that is the tool contract); responses flow back as raw JSON text.
func Register(s *mcp.Server, c *Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_articles",
		Description: "List dev.to articles. Filter by username, tag, state, or top (days). Public, no auth.",
	}, jsonTool(func(ctx context.Context, in GetArticlesInput) (json.RawMessage, error) {
		return c.GetArticles(ctx, in.query())
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_article",
		Description: "Get one article by numeric id or by path (\"username/article-slug\"). Public, no auth.",
	}, jsonTool(func(ctx context.Context, in GetArticleInput) (json.RawMessage, error) {
		switch {
		case in.ID > 0:
			return c.GetArticleByID(ctx, in.ID)
		case in.Path != "":
			return c.GetArticleByPath(ctx, in.Path)
		default:
			return nil, fmt.Errorf("provide either id or path")
		}
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_user",
		Description: "Get a user by numeric id or by username. Public, no auth.",
	}, jsonTool(func(ctx context.Context, in GetUserInput) (json.RawMessage, error) {
		switch {
		case in.ID > 0:
			return c.GetUserByID(ctx, in.ID)
		case in.Username != "":
			return c.GetUserByUsername(ctx, in.Username)
		default:
			return nil, fmt.Errorf("provide either id or username")
		}
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_tags",
		Description: "List popular dev.to tags, paginated. Public, no auth.",
	}, jsonTool(func(ctx context.Context, in PageInput) (json.RawMessage, error) {
		return c.GetTags(ctx, in.query())
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_comments",
		Description: "Get the comment tree for an article by its numeric id. Public, no auth.",
	}, jsonTool(func(ctx context.Context, in GetCommentsInput) (json.RawMessage, error) {
		if in.ArticleID <= 0 {
			return nil, fmt.Errorf("article_id must be a positive integer")
		}
		return c.GetComments(ctx, in.ArticleID)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_articles",
		Description: "Full-text search of published dev.to articles (GET /api/articles/search): q matches title, tags and body; optional top (last N days), page, per_page. Public, no auth.",
	}, jsonTool(func(ctx context.Context, in SearchInput) (json.RawMessage, error) {
		if in.Query == "" {
			return nil, fmt.Errorf("q (search query) is required")
		}
		return c.SearchArticles(ctx, in.query())
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_article",
		Description: "Create an article. Draft by default; set published=true to publish immediately. Requires API key.",
	}, jsonTool(func(ctx context.Context, in CreateArticleInput) (json.RawMessage, error) {
		return c.CreateArticle(ctx, in.article())
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_article",
		Description: "Update an existing article by id. Set published=true to publish a draft. Requires API key.",
	}, jsonTool(func(ctx context.Context, in UpdateArticleInput) (json.RawMessage, error) {
		if in.ID <= 0 {
			return nil, fmt.Errorf("id must be a positive integer")
		}
		return c.UpdateArticle(ctx, in.ID, in.article())
	}))
}

// jsonTool adapts a "give me input, get raw JSON or error" function into the
// SDK's typed handler. It is the single place that turns a client result into
// MCP text content and lets the SDK map errors to IsError (no process crash).
func jsonTool[In any](fn func(context.Context, In) (json.RawMessage, error)) mcp.ToolHandlerFor[In, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		raw, err := fn(ctx, in)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}},
		}, nil, nil
	}
}

// setInt / setStr keep query building free of repeated nil/empty checks.
func setStr(v url.Values, key, val string) {
	if val != "" {
		v.Set(key, val)
	}
}

func setInt(v url.Values, key string, val int) {
	if val > 0 {
		v.Set(key, strconv.Itoa(val))
	}
}
