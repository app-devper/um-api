package gingateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/app-devper/um-api/servicekit/gateway"
	"github.com/gin-gonic/gin"
)

func TestGinRefusesDirectAccessWithTheServicesEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware(gateway.ParseHosts("api.devper.app"), func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gateway.Message})
	}))
	reached := false
	r.GET("/x", func(c *gin.Context) { reached = true; c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Forwarded-Host", "service.a.run.app")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || reached {
		t.Fatalf("direct: %d reached=%v", rec.Code, reached)
	}

	req.Header.Set("X-Forwarded-Host", "api.devper.app")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !reached {
		t.Fatalf("gateway: %d reached=%v", rec.Code, reached)
	}
}
