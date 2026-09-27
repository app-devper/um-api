package sessionclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Role is a UM user's permission tier. UM owns the ordering:
// USER < MANAGER < ADMIN < SUPER.
type Role string

const (
	RoleUser    Role = "USER"
	RoleManager Role = "MANAGER"
	RoleAdmin   Role = "ADMIN"
	RoleSuper   Role = "SUPER"
)

var roleRank = map[Role]int{RoleUser: 1, RoleManager: 2, RoleAdmin: 3, RoleSuper: 4}

// AtLeast reports whether r is min or above. An unknown or empty role is
// below every role, and an unknown min admits nobody.
func (r Role) AtLeast(min Role) bool {
	have, ok := roleRank[r]
	need, known := roleRank[min]
	return ok && known && have >= need
}

// Principal is the verified caller behind an access token: produced only by
// verifying the token against the live UM session (see CONTEXT.md).
type Principal struct {
	SessionID string
	// UserID is empty only when DegradeReads admitted a read while UM was
	// unreachable and no session had been confirmed for this token.
	UserID   string
	Role     Role
	ClientID string
	System   string
}

// OutagePolicy decides what a request may do while UM's session store cannot
// answer. A revoked or expired session is refused under every policy.
type OutagePolicy int

const (
	// ReadOnlyWithLastGood lets a safe method (GET, HEAD, OPTIONS) continue
	// with the last session confirmed for this token; everything else waits.
	ReadOnlyWithLastGood OutagePolicy = iota
	// Strict refuses every request until the store answers: writes and
	// sensitive reads.
	Strict
	// DegradeReads lets a safe method continue under the signed token even
	// with no confirmed session: ordinary catalog reads.
	DegradeReads
)

// Error codes shared by every service that verifies UM tokens. Clients rely
// on the 503 message "identity service unavailable" to keep work pending
// instead of signing the user out.
const (
	CodeMissingToken   = "AU-401-001"
	CodeInvalidToken   = "AU-401-002"
	CodeWrongSystem    = "AU-401-003"
	CodeWrongClient    = "AU-401-004"
	CodeSessionInvalid = "AU-401-005"
	CodeForbidden      = "AU-403-002"
	CodeUnavailable    = "AU-503-001"
)

// Error is a refusal with the HTTP status, code, and message to send.
type Error struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Code + " " + e.Message + ": " + e.Err.Error()
	}
	return e.Code + " " + e.Message
}

func (e *Error) Unwrap() error { return e.Err }

func refuse(status int, code, message string, err error) *Error {
	return &Error{Status: status, Code: code, Message: message, Err: err}
}

// Forbidden is the refusal for a Principal whose role is below a route's
// minimum.
func Forbidden() *Error {
	return refuse(http.StatusForbidden, CodeForbidden, "don't have permission", nil)
}

// Config describes the tokens a service accepts.
type Config struct {
	// SecretKey is the HMAC key UM signs access tokens with.
	SecretKey string
	// System is this service's system; tokens for another system are refused.
	System string
	// ClientID, if set, is the only client whose tokens are accepted.
	ClientID string
	// Store is UM's session store: RedisStoreFor in production, a fake in tests.
	Store Store
}

// Verifier turns a request's access token into a Principal.
type Verifier struct {
	key     []byte
	system  string
	client  string
	checker *checker
}

// NewVerifier refuses an incomplete Config, so a service without UM's session
// store fails at startup instead of on every request.
func NewVerifier(cfg Config) (*Verifier, error) {
	switch {
	case cfg.SecretKey == "":
		return nil, errors.New("sessionclient: SecretKey is required")
	case cfg.System == "":
		return nil, errors.New("sessionclient: System is required")
	case cfg.Store == nil:
		return nil, errors.New("sessionclient: Store is required")
	}
	return &Verifier{
		key:     []byte(cfg.SecretKey),
		system:  cfg.System,
		client:  cfg.ClientID,
		checker: newChecker(cfg.Store),
	}, nil
}

// RedisStoreFor connects to UM's Redis at a host:port or redis:// URL. An
// empty value is an error: every service must check live sessions.
func RedisStoreFor(hostOrURL string) (Store, error) {
	if strings.TrimSpace(hostOrURL) == "" {
		return nil, errors.New("sessionclient: UM Redis host is not set")
	}
	rdb, err := NewRedisClient(hostOrURL)
	if err != nil {
		return nil, err
	}
	return NewRedisStore(rdb), nil
}

type accessClaims struct {
	Role     string `json:"role"`
	System   string `json:"system"`
	ClientId string `json:"clientId"`
	jwt.RegisteredClaims
}

// Verify checks the request's bearer token (signature, a required expiry,
// system, and a required client, pinned if configured) and the live UM
// session behind it, applying policy while the store
// cannot answer. A refusal is always an *Error.
func (v *Verifier) Verify(r *http.Request, policy OutagePolicy) (Principal, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return Principal{}, refuse(http.StatusUnauthorized, CodeMissingToken, "missing authorization header", nil)
	}
	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return v.key, nil
	}, jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.ID == "" {
		return Principal{}, refuse(http.StatusUnauthorized, CodeInvalidToken, "token invalid", err)
	}
	if claims.System != v.system {
		return Principal{}, refuse(http.StatusUnauthorized, CodeWrongSystem, "system invalid", nil)
	}
	// UM always signs a client; a token without one names no tenant.
	if claims.ClientId == "" || v.client != "" && claims.ClientId != v.client {
		return Principal{}, refuse(http.StatusUnauthorized, CodeWrongClient, "clientId invalid", nil)
	}

	p := Principal{SessionID: claims.ID, Role: Role(claims.Role), ClientID: claims.ClientId, System: claims.System}
	session, err := v.checker.Check(r.Context(), claims.ID, claims.System)
	switch {
	case err == nil:
		p.UserID = session.UserId
		return p, nil
	case errors.Is(err, ErrUnavailable):
		if admitDuringOutage(policy, r.Method, session) {
			p.UserID = session.UserId
			return p, nil
		}
		return Principal{}, refuse(http.StatusServiceUnavailable, CodeUnavailable, "identity service unavailable", err)
	default:
		return Principal{}, refuse(http.StatusUnauthorized, CodeSessionInvalid, "session invalid", err)
	}
}

func admitDuringOutage(policy OutagePolicy, method string, lastGood Session) bool {
	if !safeMethod(method) {
		return false
	}
	switch policy {
	case ReadOnlyWithLastGood:
		return lastGood.UserId != ""
	case DegradeReads:
		return true
	default:
		return false
	}
}

type principalKey struct{}

// WithPrincipal returns ctx carrying p.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom returns the Principal a Verifier middleware stored in ctx.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

// Renderer writes a refusal in the service's own response shape.
type Renderer func(w http.ResponseWriter, r *http.Request, e *Error)

// Middleware verifies every request with policy and stores the Principal in
// the request context for handlers and RequireRole.
func (v *Verifier) Middleware(policy OutagePolicy, render Renderer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, err := v.Verify(r, policy)
			if err != nil {
				render(w, r, err.(*Error))
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
		})
	}
}

// RequireRole refuses a request whose Principal is below min. It must run
// after Middleware.
func RequireRole(min Role, render Renderer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFrom(r.Context())
			if !ok || !p.Role.AtLeast(min) {
				render(w, r, Forbidden())
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
