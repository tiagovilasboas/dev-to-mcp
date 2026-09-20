// Package keychain reads secrets from the macOS Keychain via the `security`
// CLI. No CGO, no external dependency: it shells out to the same tool the OS
// ships with. On non-macOS systems the lookup simply fails and the caller
// falls back to the environment variable.
package keychain

import (
	"os/exec"
	"strings"
)

// Find returns the password stored for the given service/account in the
// macOS Keychain. It returns ok=false when the item does not exist or the
// platform is not macOS, letting the caller decide on a fallback.
func Find(service, account string) (secret string, ok bool) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", service,
		"-a", account,
		"-w", // print only the password to stdout
	)

	out, err := cmd.Output()
	if err != nil {
		return "", false
	}

	secret = strings.TrimRight(string(out), "\n")
	if secret == "" {
		return "", false
	}
	return secret, true
}
