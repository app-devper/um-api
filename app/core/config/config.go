package config

import (
	"os"
	"strings"
	"time"
)

const AccessTokenTime = 24 * time.Hour

// LoginLockoutEnabled returns whether per-username brute-force lockout is active.
// Controlled by env LOGIN_LOCKOUT_ENABLED; any of "1", "true", "yes", "on" (case-insensitive) enables it.
// Default is disabled.
func LoginLockoutEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("LOGIN_LOCKOUT_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
