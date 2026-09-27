package sessionclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testKey = "test-secret"

type tokenOpts struct {
	system, client, role, id string
	key                      string
	method                   jwt.SigningMethod
	expired, noExp, noClient bool
}

func token(t *testing.T, o tokenOpts) string {
	t.Helper()
	if o.system == "" {
		o.system = "POS"
	}
	if o.role == "" {
		o.role = "ADMIN"
	}
	if o.id == "" {
		o.id = "s1"
	}
	if o.key == "" {
		o.key = testKey
	}
	if o.method == nil {
		o.method = jwt.SigningMethodHS256
	}
	if o.client == "" && !o.noClient {
		o.client = "001"
	}
	exp := time.Now().Add(time.Hour)
	if o.expired {
		exp = time.Now().Add(-time.Minute)
	}
	claims := accessClaims{Role: o.role, System: o.system, ClientId: o.client,
		RegisteredClaims: jwt.RegisteredClaims{ID: o.id, ExpiresAt: jwt.NewNumericDate(exp)}}
	if o.noExp {
		claims.ExpiresAt = nil
	}
	var key interface{} = []byte(o.key)
	if o.method == jwt.SigningMethodNone {
		key = jwt.UnsafeAllowNoneSignatureType
	}
	signed, err := jwt.NewWithClaims(o.method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func verifier(t *testing.T, store Store, client string) *Verifier {
	t.Helper()
	v, err := NewVerifier(Config{SecretKey: testKey, System: "POS", ClientID: client, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func request(method, bearer string) *http.Request {
	r := httptest.NewRequest(method, "/", nil)
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	return r
}

func wantRefusal(t *testing.T, err error, status int, code string) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Status != status || e.Code != code {
		t.Fatalf("expected %d %s, got %v", status, code, err)
	}
}

func TestVerifyReturnsPrincipalForLiveSession(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	p, err := verifier(t, store, "").Verify(request(http.MethodPost, token(t, tokenOpts{client: "001", role: "MANAGER"})), Strict)
	if err != nil {
		t.Fatal(err)
	}
	want := Principal{SessionID: "s1", UserID: "u1", Role: RoleManager, ClientID: "001", System: "POS"}
	if p != want {
		t.Fatalf("got %+v, want %+v", p, want)
	}
}

func TestVerifyRefusesBadTokens(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	cases := map[string]struct {
		header string
		status int
		code   string
	}{
		"missing header": {"", http.StatusUnauthorized, CodeMissingToken},
		"not bearer":     {"Basic abc", http.StatusUnauthorized, CodeMissingToken},
		"wrong key":      {"Bearer " + token(t, tokenOpts{key: "other"}), http.StatusUnauthorized, CodeInvalidToken},
		"alg none":       {"Bearer " + token(t, tokenOpts{method: jwt.SigningMethodNone}), http.StatusUnauthorized, CodeInvalidToken},
		"expired":        {"Bearer " + token(t, tokenOpts{expired: true}), http.StatusUnauthorized, CodeInvalidToken},
		"another system": {"Bearer " + token(t, tokenOpts{system: "GOLD"}), http.StatusUnauthorized, CodeWrongSystem},
		"another client": {"Bearer " + token(t, tokenOpts{client: "002"}), http.StatusUnauthorized, CodeWrongClient},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				r.Header.Set("Authorization", tc.header)
			}
			_, err := verifier(t, store, "001").Verify(r, ReadOnlyWithLastGood)
			wantRefusal(t, err, tc.status, tc.code)
		})
	}
}

func TestVerifyRefusesRevokedOrForeignSession(t *testing.T) {
	for name, store := range map[string]*fakeStore{
		"revoked":        {err: ErrSessionRejected},
		"another system": {session: Session{UserId: "u1", System: "GOLD"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := verifier(t, store, "").Verify(request(http.MethodGet, token(t, tokenOpts{})), DegradeReads)
			wantRefusal(t, err, http.StatusUnauthorized, CodeSessionInvalid)
		})
	}
}

// Outage policies: what each lets through while the store cannot answer.
func TestVerifyAppliesOutagePolicy(t *testing.T) {
	cases := []struct {
		name     string
		policy   OutagePolicy
		method   string
		lastGood bool
		admitted bool
	}{
		{"read-only: GET with last good", ReadOnlyWithLastGood, http.MethodGet, true, true},
		{"read-only: GET without last good", ReadOnlyWithLastGood, http.MethodGet, false, false},
		{"read-only: POST with last good", ReadOnlyWithLastGood, http.MethodPost, true, false},
		{"strict: GET with last good", Strict, http.MethodGet, true, false},
		{"degrade: GET without last good", DegradeReads, http.MethodGet, false, true},
		{"degrade: POST", DegradeReads, http.MethodPost, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
			v := verifier(t, store, "")
			tok := token(t, tokenOpts{})
			if tc.lastGood {
				if _, err := v.Verify(request(http.MethodGet, tok), Strict); err != nil {
					t.Fatal(err)
				}
			}
			store.err = ErrUnavailable
			v.checker.ttl = 0 // force a lookup

			p, err := v.Verify(request(tc.method, tok), tc.policy)
			if !tc.admitted {
				wantRefusal(t, err, http.StatusServiceUnavailable, CodeUnavailable)
				if err.(*Error).Message != "identity service unavailable" {
					t.Fatalf("clients rely on the 503 message, got %q", err.(*Error).Message)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected admission, got %v", err)
			}
			if tc.lastGood && p.UserID != "u1" {
				t.Fatalf("expected last-good user, got %+v", p)
			}
		})
	}
}

func TestNewVerifierRefusesIncompleteConfig(t *testing.T) {
	store := &fakeStore{}
	for name, cfg := range map[string]Config{
		"no key":    {System: "POS", Store: store},
		"no system": {SecretKey: testKey, Store: store},
		"no store":  {SecretKey: testKey, System: "POS"},
	} {
		if _, err := NewVerifier(cfg); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if _, err := RedisStoreFor("  "); err == nil {
		t.Error("expected an error for an unset Redis host")
	}
}

func TestRoleAtLeastFollowsUMOrdering(t *testing.T) {
	order := []Role{RoleUser, RoleManager, RoleAdmin, RoleSuper}
	for i, r := range order {
		for j, min := range order {
			if got := r.AtLeast(min); got != (i >= j) {
				t.Errorf("%s.AtLeast(%s) = %v", r, min, got)
			}
		}
	}
	for _, r := range []Role{"", "STAFF", "admin"} {
		if r.AtLeast(RoleUser) {
			t.Errorf("unknown role %q must be below every role", r)
		}
	}
	if RoleSuper.AtLeast("STAFF") {
		t.Error("an unknown minimum must admit nobody")
	}
}

func TestMiddlewareStoresPrincipalAndRequireRoleGates(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	v := verifier(t, store, "")
	var refused *Error
	render := func(w http.ResponseWriter, _ *http.Request, e *Error) { refused = e; w.WriteHeader(e.Status) }
	var seen Principal
	h := v.Middleware(Strict, render)(RequireRole(RoleAdmin, render)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen, _ = PrincipalFrom(r.Context())
	})))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, request(http.MethodGet, token(t, tokenOpts{role: "SUPER"})))
	if w.Code != http.StatusOK || seen.UserID != "u1" {
		t.Fatalf("expected SUPER through, got %d %+v", w.Code, seen)
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, request(http.MethodGet, token(t, tokenOpts{role: "MANAGER", id: "s2"})))
	if w.Code != http.StatusForbidden || refused.Code != CodeForbidden {
		t.Fatalf("expected 403 %s for MANAGER, got %d %+v", CodeForbidden, w.Code, refused)
	}
}
