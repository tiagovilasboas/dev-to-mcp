package keychain

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fakeSecurity puts a stub `security` binary first on PATH so Find can be
// exercised on any OS without touching a real Keychain.
func fakeSecurity(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell stub not supported on windows")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "security"), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestFindReturnsTrimmedSecret(t *testing.T) {
	fakeSecurity(t, `[ "$1 $2 $3 $4 $5 $6" = "find-generic-password -s svc -a acct -w" ] || exit 44
printf 'tok123\n'`)
	got, ok := Find("svc", "acct")
	if !ok || got != "tok123" {
		t.Errorf("Find = %q, %v", got, ok)
	}
}

func TestFindMissingItem(t *testing.T) {
	fakeSecurity(t, `exit 44`)
	if got, ok := Find("svc", "acct"); ok || got != "" {
		t.Errorf("Find = %q, %v; want not found", got, ok)
	}
}

func TestFindEmptySecret(t *testing.T) {
	fakeSecurity(t, `printf '\n'`)
	if _, ok := Find("svc", "acct"); ok {
		t.Error("empty secret must be reported as not found")
	}
}

func TestFindWithoutSecurityBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, ok := Find("svc", "acct"); ok {
		t.Error("missing security CLI must be reported as not found")
	}
}
