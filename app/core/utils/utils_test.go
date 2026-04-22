package utils

import "testing"

func TestNormalizeUsernameTrimsWhitespace(t *testing.T) {
	got := NormalizeUsername("  alice  ")
	if got != "alice" {
		t.Fatalf("expected trimmed username, got %q", got)
	}
}
