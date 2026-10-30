package errs

const (
	// 401 Unauthorized
	ErrMissingAuthHeader   = "UM-401-001"
	ErrTokenInvalid        = "UM-401-002"
	ErrSessionInvalid      = "UM-401-003"
	ErrWrongCredentials    = "UM-401-004"
	ErrPasskeyRejected     = "UM-401-005" // the authenticator's answer did not verify
	ErrPasskeyOtherAccount = "UM-401-006" // the passkey verified but belongs to a different User than the typed username

	// 400 Bad Request
	ErrBadRequest      = "UM-400-001"
	ErrWrongPassword   = "UM-400-002"
	ErrInvalidClientId = "UM-400-003"
	ErrInvalidRole     = "UM-400-004"
	ErrDeleteSelf      = "UM-400-005"
	ErrCeremonyExpired = "UM-400-006" // a passkey request expired or was already answered; start again

	// 403 Forbidden
	ErrForbidden             = "UM-403-001"
	ErrNoPermission          = "UM-403-002"
	ErrInvalidRolePermission = "UM-403-003"

	// 409 Conflict
	ErrUsernameTaken = "UM-409-001"
	ErrSystemInUse   = "UM-409-002" // a catalog System that Clients still have

	// 404 Not Found
	ErrNotFound         = "UM-404-001"
	ErrPasskeysDisabled = "UM-404-002" // WEBAUTHN_RP_ID is not set

	// 429 Too Many Requests
	ErrRateLimited = "UM-429-001"

	// 503 Service Unavailable
	ErrUnavailable      = "UM-503-001" // the lockout or session store cannot be reached
	ErrSessionsNotEnded = "UM-503-002" // a change was saved but the user's Sessions could not be ended; retry

	// 500 Internal Server Error
	ErrInternal       = "UM-500-001"
	ErrTokenGenFailed = "UM-500-002"
)
