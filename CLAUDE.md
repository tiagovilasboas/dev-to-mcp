# CLAUDE.md

Guidance for AI agents working in this repository.

## What this is

A Go MCP server for the dev.to API, spoken over **stdio**. It replaced an earlier Node/HTTP version. Do not reintroduce Node, Express, an HTTP transport, or a listening port.

## Commands

```bash
go build -o dist/dev-to-mcp .   # build the binary
go vet ./...                    # static checks
go test ./...                   # tests (add as needed)
```

## Architecture

Layered, one responsibility per file:

- `main.go` — bootstrap: resolve the API token, build the `mcp.Server`, run it over `StdioTransport`.
- `internal/keychain/keychain.go` — read a secret from the macOS Keychain via the `security` CLI (no CGO).
- `internal/devto/client.go` — HTTP transport only: `get`, `writeArticle`, `do`. Maps non-2xx to errors carrying the body.
- `internal/devto/endpoints.go` — one small method per dev.to endpoint; builds path/query and delegates.
- `internal/devto/tools.go` — registers the 8 MCP tools; `jsonTool` adapts a client call into the SDK's typed handler.
- `internal/devto/inputs.go` — typed tool inputs with `jsonschema` tags; each owns its query/body builder.

## Conventions

- **SDK:** official `github.com/modelcontextprotocol/go-sdk`. Register tools with `mcp.AddTool` and typed input structs.
- **Responses are raw JSON.** The client returns `json.RawMessage`; tools wrap it in `TextContent`. Do not add response structs — the consumer is an LLM that reads JSON, and the upstream shape changes.
- **Type the input, not the output.** Input structs are the tool contract; `jsonschema` tags are the schema shown to the model.
- **Errors never crash the process.** Return an `error` from a handler and the SDK marks `isError`. Never `log.Fatal` inside a tool.
- **Logs to stderr only.** stdout is the JSON-RPC channel. Keep it clean.
- **No goroutines** unless a tool genuinely fans out (then use `errgroup`). One request in, one HTTP call, one response.
- **DRY:** reads share `get`; writes share `writeArticle`; create/update share `articleFields`.

## API key

Resolved Keychain first, then `DEV_TO_API_KEY`. The Keychain service and account come from `DEVTO_KEYCHAIN_SERVICE` / `DEVTO_KEYCHAIN_ACCOUNT`, defaulting to `dev-to-mcp` and the current OS user — nothing is hardcoded to a person. Never log the token value. Never commit it.

## Endpoint facts (dev.to / Forem)

- Base URL `https://dev.to/api/`.
- Writes: header `api-key: <token>` + `Accept: application/vnd.forem.api-v1+json`, body `{"article": {...}}`.
- Create = `POST /articles`; update/publish = `PUT /articles/{id}`; publish = `published: true`.
- Reads are public (no auth).
