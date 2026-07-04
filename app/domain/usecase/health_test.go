package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func healthCall(t *testing.T, mongoPing, redisPing Pinger) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	Health(mongoPing, redisPing)(c)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v (raw=%s)", err, w.Body.String())
	}
	return w, body
}

func TestHealthOkWhenBothPingsSucceed(t *testing.T) {
	okPing := func(context.Context) error { return nil }

	w, body := healthCall(t, okPing, okPing)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %v", body["status"])
	}
	checks, _ := body["checks"].(map[string]any)
	if checks["mongo"] != "ok" || checks["redis"] != "ok" {
		t.Fatalf("expected both checks ok, got %+v", checks)
	}
}

func TestHealthDegradedWhenMongoFails(t *testing.T) {
	mongoPing := func(context.Context) error { return errors.New("mongo is down") }
	redisPing := func(context.Context) error { return nil }

	w, body := healthCall(t, mongoPing, redisPing)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
	if body["status"] != "degraded" {
		t.Fatalf("expected status=degraded, got %v", body["status"])
	}
	checks, _ := body["checks"].(map[string]any)
	if checks["mongo"] != "mongo is down" {
		t.Fatalf("expected mongo error reported, got %v", checks["mongo"])
	}
	if checks["redis"] != "ok" {
		t.Fatalf("expected redis ok, got %v", checks["redis"])
	}
}

func TestHealthDegradedWhenRedisFails(t *testing.T) {
	mongoPing := func(context.Context) error { return nil }
	redisPing := func(context.Context) error { return errors.New("redis is down") }

	w, body := healthCall(t, mongoPing, redisPing)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
	if body["status"] != "degraded" {
		t.Fatalf("expected status=degraded, got %v", body["status"])
	}
	checks, _ := body["checks"].(map[string]any)
	if checks["mongo"] != "ok" {
		t.Fatalf("expected mongo ok, got %v", checks["mongo"])
	}
	if checks["redis"] != "redis is down" {
		t.Fatalf("expected redis error reported, got %v", checks["redis"])
	}
}

func TestHealthDegradedWhenBothFail(t *testing.T) {
	mongoPing := func(context.Context) error { return errors.New("mongo is down") }
	redisPing := func(context.Context) error { return errors.New("redis is down") }

	w, body := healthCall(t, mongoPing, redisPing)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
	if body["status"] != "degraded" {
		t.Fatalf("expected status=degraded, got %v", body["status"])
	}
}

func TestHealthPassesDeadlineToPingers(t *testing.T) {
	var sawMongoDeadline, sawRedisDeadline bool
	mongoPing := func(ctx context.Context) error {
		_, ok := ctx.Deadline()
		sawMongoDeadline = ok
		return nil
	}
	redisPing := func(ctx context.Context) error {
		_, ok := ctx.Deadline()
		sawRedisDeadline = ok
		return nil
	}

	w, _ := healthCall(t, mongoPing, redisPing)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !sawMongoDeadline || !sawRedisDeadline {
		t.Fatalf("expected pinger ctx to carry a deadline, mongo=%v redis=%v", sawMongoDeadline, sawRedisDeadline)
	}
}
