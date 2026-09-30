package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnvOr(t *testing.T) {
	t.Setenv("DEVTO_TEST_ENV", "")
	if got := envOr("DEVTO_TEST_ENV", "fb"); got != "fb" {
		t.Errorf("empty env: got %q", got)
	}
	t.Setenv("DEVTO_TEST_ENV", "set")
	if got := envOr("DEVTO_TEST_ENV", "fb"); got != "set" {
		t.Errorf("set env: got %q", got)
	}
}

func TestCurrentUserIsNotEmpty(t *testing.T) {
	if currentUser() == "" {
		t.Skip("no OS user in this environment")
	}
}

// stubSecurity installs a fake `security` CLI that prints secret only for the
// expected service/account pair.
func stubSecurity(t *testing.T, service, account, secret string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell stub not supported on windows")
	}
	dir := t.TempDir()
	script := "#!/bin/sh\n" +
		`[ "$3" = "` + service + `" ] && [ "$5" = "` + account + `" ] || exit 44` + "\n" +
		`printf '%s\n' "` + secret + `"` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "security"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestResolveTokenPrefersKeychain(t *testing.T) {
	stubSecurity(t, "svc", "acct", "from-keychain")
	t.Setenv("DEVTO_KEYCHAIN_SERVICE", "svc")
	t.Setenv("DEVTO_KEYCHAIN_ACCOUNT", "acct")
	t.Setenv("DEV_TO_API_KEY", "from-env")
	if got := resolveToken(); got != "from-keychain" {
		t.Errorf("resolveToken = %q", got)
	}
}

func TestResolveTokenFallsBackToEnv(t *testing.T) {
	stubSecurity(t, "svc", "acct", "from-keychain")
	t.Setenv("DEVTO_KEYCHAIN_SERVICE", "other")
	t.Setenv("DEVTO_KEYCHAIN_ACCOUNT", "acct")
	t.Setenv("DEV_TO_API_KEY", "from-env")
	if got := resolveToken(); got != "from-env" {
		t.Errorf("resolveToken = %q", got)
	}
}

func TestResolveTokenDefaultsService(t *testing.T) {
	stubSecurity(t, defaultKeychainService, "acct", "default-svc")
	t.Setenv("DEVTO_KEYCHAIN_SERVICE", "")
	t.Setenv("DEVTO_KEYCHAIN_ACCOUNT", "acct")
	t.Setenv("DEV_TO_API_KEY", "")
	if got := resolveToken(); got != "default-svc" {
		t.Errorf("resolveToken = %q", got)
	}
}
