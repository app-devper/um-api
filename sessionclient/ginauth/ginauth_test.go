package ginauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/app-devper/um-api/sessionclient"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type store struct{ err error }

func (s store) Session(context.Context, string) (sessionclient.Session, error) {
	return sessionclient.Session{UserId: "u1", System: "POS"}, s.err
}

func signed(t *testing.T, role string) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"role": role, "system": "POS", "clientId": "001", "jti": "s1",
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("k"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func router(t *testing.T, s sessionclient.Store, min sessionclient.Role, handler gin.HandlerFunc) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	v, err := sessionclient.NewVerifier(sessionclient.Config{SecretKey: "k", System: "POS", Store: s})
	if err != nil {
		t.Fatal(err)
	}
	auth := New(v, func(c *gin.Context, e *sessionclient.Error) {
		c.JSON(e.Status, gin.H{"code": e.Code, "error": e.Message})
	})
	r := gin.New()
	r.GET("/p", auth.Require(sessionclient.Strict), auth.AtLeast(min), handler)
	return r
}

func get(r *gin.Engine, bearer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRequireSetsPrincipalAndLegacyKeys(t *testing.T) {
	var keys []string
	var p sessionclient.Principal
	r := router(t, store{}, sessionclient.RoleManager, func(c *gin.Context) {
		p, _ = Principal(c)
		for _, k := range []string{"SessionId", "UserId", "Role", "ClientId", "System"} {
			keys = append(keys, c.GetString(k))
		}
		if _, ok := sessionclient.PrincipalFrom(c.Request.Context()); !ok {
			t.Error("expected Principal in the request context")
		}
		c.Status(http.StatusOK)
	})
	if w := get(r, signed(t, "ADMIN")); w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", w.Code, w.Body.String())
	}
	if p.UserID != "u1" || p.Role != sessionclient.RoleAdmin {
		t.Fatalf("unexpected principal %+v", p)
	}
	want := []string{"s1", "u1", "ADMIN", "001", "POS"}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("legacy keys = %v, want %v", keys, want)
		}
	}
}

func TestRequireRendersRefusalsAndAborts(t *testing.T) {
	called := false
	r := router(t, store{err: sessionclient.ErrUnavailable}, sessionclient.RoleUser, func(c *gin.Context) { called = true })
	if w := get(r, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if w := get(r, signed(t, "ADMIN")); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 under Strict, got %d", w.Code)
	}
	if called {
		t.Fatal("handler must not run after a refusal")
	}
}

func TestAtLeastRefusesLowerRole(t *testing.T) {
	r := router(t, store{}, sessionclient.RoleAdmin, func(c *gin.Context) { c.Status(http.StatusOK) })
	w := get(r, signed(t, "MANAGER"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d %s", w.Code, w.Body.String())
	}
}
