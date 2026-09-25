package errs

import (
	"net/http"
	"testing"
)

func TestStatusComesFromCode(t *testing.T) {
	cases := map[string]int{
		ErrNoPermission:      http.StatusForbidden,
		ErrUsernameTaken:     http.StatusConflict,
		ErrMissingAuthHeader: http.StatusUnauthorized,
		ErrTokenGenFailed:    http.StatusInternalServerError,
		"UM-999-001":         http.StatusInternalServerError,
		"garbage":            http.StatusInternalServerError,
	}
	for code, want := range cases {
		if got := New(code, "").Status(); got != want {
			t.Errorf("%s: expected %d, got %d", code, want, got)
		}
	}
}
