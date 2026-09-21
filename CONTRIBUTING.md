# Contributing to dev-to-mcp

Thanks for your interest in contributing! This project is small and focused, so the process is lightweight.

## Ways to contribute

- **Report bugs** — open an issue describing what happened and what you expected
- **Suggest features** — open an issue explaining the use case
- **Submit a fix or feature** — fork, code, test, PR

## Development setup

Requires Go 1.27+.

```bash
git clone https://github.com/tiagovilasboas/dev-to-mcp.git
cd dev-to-mcp
go build -o dist/dev-to-mcp .
go test ./...
```

### Project structure

```
main.go                        bootstrap: resolve token, build server, run stdio
internal/keychain/keychain.go  macOS Keychain integration
internal/devto/
├── client.go                  HTTP transport
├── endpoints.go               one method per dev.to endpoint
├── tools.go                   MCP tool definitions
└── inputs.go                  typed tool inputs
```

### Testing locally

1. Build the binary: `go build -o dist/dev-to-mcp .`
2. Point your MCP client at `dist/dev-to-mcp`
3. Test the tools via your client (Kiro, Cursor, Claude Desktop, etc.)

For write tools, you'll need a dev.to API key — get one at [dev.to/settings/extensions](https://dev.to/settings/extensions).

## Submitting a pull request

1. **Open an issue first** (for non-trivial changes) — let's discuss the approach before you invest time coding
2. **Fork and branch** — create a feature branch from `main`
3. **Keep it focused** — one PR per feature or fix
4. **Test your changes** — run `go test ./...` and manually test the affected tools
5. **Write clear commits** — use imperative mood (`Add feature`, not `Added feature`)

### Commit message format

```
<type>: <short summary>

<optional body>
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

Examples:
- `feat: add get_followers tool`
- `fix: handle rate limit response from dev.to`
- `docs: clarify Keychain setup on macOS`

## Code style

- Follow standard Go conventions (`go fmt`, `go vet`)
- Keep it simple — this is a small, focused project
- One responsibility per file
- Responses pass through as raw JSON — no response structs to maintain

## Questions?

Open an issue or start a discussion. We're friendly.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
