package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"um/app/core/errs"
	"um/app/domain/session"
	"um/app/domain/usecase"
	"um/app/domain/useradmin"

	"github.com/gin-gonic/gin"
)

// Every route except these must reject a request without a Session, so a
// stale JWT can never reach a handler.
var publicRoutes = map[string]bool{
	"POST /api/um/v1/auth/login":    true,
	"POST /api/um/v1/auth/exchange": true,
}

func TestEveryNonPublicRouteRequiresSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	public := r.Group("/api/um/v1")
	sessions := session.NewManager("test-secret", nil, nil)
	protected := public.Group("", usecase.RequireSession(sessions))

	ApplyAuthAPI(public, protected, sessions, nil, nil, nil, nil, nil, nil)
	ApplyUserAPI(protected, nil, useradmin.New(nil, nil, nil, nil))
	ApplySystemAPI(protected, nil)

	routes := r.Routes()
	if len(routes) < 25 {
		t.Fatalf("expected the full route table, got %d routes", len(routes))
	}
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if publicRoutes[key] {
			continue
		}
		w := httptest.NewRecorder()
		path := strings.ReplaceAll(route.Path, ":id", "0123456789abcdef01234567")
		r.ServeHTTP(w, httptest.NewRequest(route.Method, path, nil))
		if w.Code != http.StatusUnauthorized || !bytes.Contains(w.Body.Bytes(), []byte(errs.ErrMissingAuthHeader)) {
			t.Errorf("%s: expected 401 %s without a Session, got %d body=%s", key, errs.ErrMissingAuthHeader, w.Code, w.Body.String())
		}
	}
}
