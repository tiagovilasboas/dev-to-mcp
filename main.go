// Command dev-to-mcp is a Model Context Protocol server for the dev.to API,
// spoken over stdio. The client launches it on demand; there is no port and no
// long-lived HTTP server to fall over. Logs go to stderr so stdout stays a
// clean JSON-RPC channel.
package main

import (
	"context"
	"log"
	"os"
	"os/user"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tiagovilasboas/dev-to-mcp/internal/devto"
	"github.com/tiagovilasboas/dev-to-mcp/internal/keychain"
)

const version = "1.0.0"

// defaultKeychainService is overridable via DEVTO_KEYCHAIN_SERVICE so the tool
// is not pinned to any single account. Nothing here is hardcoded to a person.
const defaultKeychainService = "dev-to-mcp"

func main() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("dev-to-mcp: ")

	token := resolveToken()
	if token == "" {
		log.Println("no API key found (Keychain or DEV_TO_API_KEY): read tools work, write tools will fail")
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "dev-to-mcp", Version: version}, nil)
	devto.Register(server, devto.New(token))

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

// resolveToken prefers the macOS Keychain and falls back to DEV_TO_API_KEY.
// The Keychain service and account come from env vars with neutral defaults:
// the account defaults to the current OS user, so nothing is tied to a person.
func resolveToken() string {
	service := envOr("DEVTO_KEYCHAIN_SERVICE", defaultKeychainService)
	account := envOr("DEVTO_KEYCHAIN_ACCOUNT", currentUser())

	if secret, ok := keychain.Find(service, account); ok {
		return secret
	}
	return os.Getenv("DEV_TO_API_KEY")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func currentUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return ""
}
