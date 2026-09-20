# Contributing

Thanks for your interest in improving **dev-to-mcp**.

This is an AI-assisted project: code is pair-programmed with an agent and then reviewed and tested by a human. Contributions are held to the same bar — reviewed and tested before merge.

## Code of Conduct

This project is governed by our [Code of Conduct](CODE-OF-CONDUCT.md). By participating, you agree to uphold it. Report unacceptable behavior to the maintainer at tcarvalhovb@gmail.com or by opening a confidential issue.

## Getting started

Requires Go 1.27+.

```bash
git clone https://github.com/tiagovilasboas/dev-to-mcp.git
cd dev-to-mcp
go build -o dist/dev-to-mcp .
```

## Making a change

1. Fork the repo and create a branch from `main`.
2. Make your change, keeping the layout's one-responsibility-per-file convention (see [CLAUDE.md](CLAUDE.md)).
3. Run the quality checks:
   ```bash
   go vet ./...
   go build ./...
   go test ./...   # if you added tests
   ```
4. Do not hardcode secrets or personal identifiers. The API key is resolved from the Keychain or `DEV_TO_API_KEY`; never commit a key.
5. Open a pull request describing what changed and how you verified it.

## Reporting bugs

Open an issue with steps to reproduce, the tool you called, and the observed vs expected result. Never paste your dev.to API key into an issue.

## Questions

Open an issue for discussion.
