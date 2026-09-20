<img src=".github/cover.png" alt="dev-to-mcp" width="100%">

# dev-to-mcp

> A fast, single-binary **Model Context Protocol (MCP)** server for the [dev.to](https://dev.to) API, written in Go and spoken over **stdio**.

![Language](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go&logoColor=white)
![MCP](https://img.shields.io/badge/MCP-official%20go--sdk-5A45FF)
![Transport](https://img.shields.io/badge/transport-stdio-2ea44f)
![License](https://img.shields.io/badge/license-MIT-blue)

The MCP client (Kiro, Cursor, Claude Desktop, VS Code, ...) launches the binary on demand and talks to it over stdin/stdout. There is no port and no long-lived HTTP server to fall over.

> [!NOTE]
> **AI-assisted project.** This server was designed and implemented with AI assistance (pair-programmed with an agent), then reviewed and tested by a human. Every design decision, security choice, and the code itself were verified before landing. Contributions are welcome under the same bar: reviewed and tested.

---

## Table of contents

- [Why Go + stdio](#why-go--stdio)
- [Tools](#tools)
- [Install](#install)
- [Get a dev.to API key](#get-a-devto-api-key)
- [Store the key securely](#store-the-key-securely)
- [Configure your MCP client](#configure-your-mcp-client)
- [Security](#security)
- [Configuration reference](#configuration-reference)
- [Project layout](#project-layout)
- [License](#license)

---

## Why Go + stdio

- **Single binary.** No `npx` or `node_modules` download at startup, so no cold-start timeout or network flakiness when the client spawns it.
- **stdio, not HTTP.** The client owns the process. No `:3000` server pinned in the background, no session bookkeeping.
- **Errors never crash the process.** A failed dev.to call becomes a tool error (`isError: true`) the model can read and retry, not a dead server.
- **Logs go to stderr.** stdout stays a clean JSON-RPC channel.

Built on the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).

## Tools

| Tool | Auth | Description |
|------|:----:|-------------|
| `get_articles` | public | List articles; filter by username, tag, state, or top (days) |
| `get_article` | public | One article by numeric `id` or `path` (`username/article-slug`) |
| `get_user` | public | User by `id` or `username` |
| `get_tags` | public | Popular tags, paginated |
| `get_comments` | public | Comment tree for an `article_id` |
| `search_articles` | public | Search articles by query |
| `create_article` | 🔑 key | Create an article (draft by default) |
| `update_article` | 🔑 key | Update an article by `id` |

Publishing is not a separate tool: it is `create_article` / `update_article` with `published: true`. Read tools work with no key at all.

## Install

Requires Go 1.27+.

```bash
git clone https://github.com/tiagovilasboas/dev-to-mcp.git
cd dev-to-mcp
go build -o dist/dev-to-mcp .
```

This produces a self-contained binary at `dist/dev-to-mcp`. Note the absolute path — you will point your MCP client at it.

## Get a dev.to API key

The write tools (`create_article`, `update_article`) need a personal API key. Generate one at **dev.to → Settings → Extensions → DEV Community API Keys** ([direct link](https://dev.to/settings/extensions)). Read tools need nothing.

## Store the key securely

**Never put your key in a config file that can be committed to git.** Pick the option that fits your OS.

### macOS — Keychain (recommended, natively supported)

The binary reads the key straight from the macOS Keychain, so it lives in the OS secret store and never touches a config file:

```bash
security add-generic-password -s dev-to-mcp -a "$(id -un)" -w "YOUR_DEV_TO_API_KEY" -U
```

That is it. The server finds it automatically (service `dev-to-mcp`, account = your OS user).

### Linux / Windows — environment via a vault

Native Keychain lookup is macOS-only. On Linux/Windows, feed the key through the `DEV_TO_API_KEY` environment variable, sourced from a secret manager rather than typed in plain text — for example:

- **1Password CLI:** `op run -- dist/dev-to-mcp` with `DEV_TO_API_KEY=op://vault/dev-to/key`
- **`pass`:** `DEV_TO_API_KEY=$(pass dev-to/api-key) dist/dev-to-mcp`
- your MCP client's own encrypted secret store, if it has one

Passing the key inline in an `mcp.json` `env` block works, but only do it if that file is **git-ignored** and you accept the local-plaintext trade-off.

## Configure your MCP client

Point the client at the compiled binary using its **absolute path**. If your key is in the macOS Keychain, no `env` block is needed.

<details open>
<summary><b>Kiro</b> — <code>.kiro/settings/mcp.json</code> (workspace) or <code>~/.kiro/settings/mcp.json</code> (global)</summary>

```json
{
  "mcpServers": {
    "dev-to": {
      "command": "/absolute/path/to/dev-to-mcp/dist/dev-to-mcp",
      "disabled": false
    }
  }
}
```
</details>

<details>
<summary><b>Cursor</b> — <code>~/.cursor/mcp.json</code> or <code>.cursor/mcp.json</code></summary>

```json
{
  "mcpServers": {
    "dev-to": {
      "command": "/absolute/path/to/dev-to-mcp/dist/dev-to-mcp"
    }
  }
}
```
</details>

<details>
<summary><b>Claude Desktop</b> — <code>claude_desktop_config.json</code></summary>

```json
{
  "mcpServers": {
    "dev-to": {
      "command": "/absolute/path/to/dev-to-mcp/dist/dev-to-mcp"
    }
  }
}
```
</details>

<details>
<summary><b>VS Code</b> (Continue / Cline and other MCP extensions)</summary>

```json
{
  "mcpServers": {
    "dev-to": {
      "command": "/absolute/path/to/dev-to-mcp/dist/dev-to-mcp"
    }
  }
}
```
</details>

If you must pass the key via environment (Linux/Windows, or no vault), add an `env` block to any of the above — and keep that file out of git:

```json
{
  "mcpServers": {
    "dev-to": {
      "command": "/absolute/path/to/dev-to-mcp/dist/dev-to-mcp",
      "env": { "DEV_TO_API_KEY": "YOUR_DEV_TO_API_KEY" }
    }
  }
}
```

## Security

This server is designed to keep your credentials and data safe. What it does:

- **Vault-first key handling.** On macOS the key is read from the Keychain; it is never written to a config file by this project.
- **The token is never logged.** Only diagnostics go to stderr, and the key value is not among them.
- **No key in the repo.** Nothing is hardcoded to a user; the Keychain account defaults to the current OS user and is overridable by env var.
- **Token pass-through only.** The server sends your key to `https://dev.to` and nowhere else. It does not decode, inspect, or store it.
- **Responses pass through unchanged.** dev.to's JSON is returned as-is; no third-party endpoint is contacted.

What you should do:

- Keep your key in a vault (Keychain / 1Password / `pass`), not in a committed file.
- Ensure any file containing `DEV_TO_API_KEY` is git-ignored.
- Rotate the key at dev.to if you suspect it was exposed.

> Write tools (`create_article`, `update_article`) publish to your real dev.to account with **no extra confirmation** — approval is expected to come from your MCP client's tool-approval prompt. Review calls before you approve them.

## Configuration reference

All optional. Sensible defaults mean macOS users configure nothing.

| Variable | Default | Purpose |
|----------|---------|---------|
| `DEV_TO_API_KEY` | — | API key fallback when the Keychain has no entry (required on Linux/Windows for write tools) |
| `DEVTO_KEYCHAIN_SERVICE` | `dev-to-mcp` | macOS Keychain service name to read |
| `DEVTO_KEYCHAIN_ACCOUNT` | current OS user | macOS Keychain account name to read |

Resolution order for the key: **Keychain** (`service` + `account`) → **`DEV_TO_API_KEY`**.

## Project layout

```
main.go                        bootstrap: resolve token, build server, run stdio
internal/keychain/keychain.go  read a secret from the macOS Keychain (no CGO)
internal/devto/
  client.go                    HTTP transport: get / writeArticle / do
  endpoints.go                 one method per dev.to endpoint
  tools.go                     register the 8 MCP tools
  inputs.go                    typed tool inputs + query/body builders
```

One responsibility per file. Responses flow back as raw dev.to JSON — the consumer is an LLM that reads JSON, so there are no response structs to keep in sync with the upstream API.

## License

MIT
