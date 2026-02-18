package errs

const (
	// 401 Unauthorized
	ErrMissingAuthHeader = "UM-401-001"
	ErrTokenInvalid      = "UM-401-002"
	ErrSessionInvalid    = "UM-401-003"
	ErrWrongCredentials  = "UM-401-004"

	// 400 Bad Request
	ErrBadRequest      = "UM-400-001"
	ErrWrongPassword   = "UM-400-002"
	ErrInvalidClientId = "UM-400-003"
	ErrInvalidRole     = "UM-400-004"
	ErrDeleteSelf      = "UM-400-005"

	// 403 Forbidden
	ErrForbidden             = "UM-403-001"
	ErrNoPermission          = "UM-403-002"
	ErrInvalidRolePermission = "UM-403-003"

	// 409 Conflict
	ErrUsernameTaken = "UM-409-001"

	// 404 Not Found
	ErrNotFound = "UM-404-001"

	// 429 Too Many Requests
	ErrRateLimited = "UM-429-001"

	// 500 Internal Server Error
	ErrInternal       = "UM-500-001"
	ErrTokenGenFailed = "UM-500-002"
)
