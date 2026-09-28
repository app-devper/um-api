package middlewares

import (
	"net/http"

	"um/app/core/errs"

	"github.com/app-devper/um-api/servicekit/gateway"
	"github.com/gin-gonic/gin"
)

// NewGatewayHost refuses requests that did not come through the gateway
// (servicekit/gateway, ADR-0007), in UM's error envelope.
func NewGatewayHost(allowedHosts string) gin.HandlerFunc {
	hosts := gateway.ParseHosts(allowedHosts)
	return func(c *gin.Context) {
		if !hosts.Allows(c.Request) {
			errs.Response(c, http.StatusForbidden, errs.New(errs.ErrForbidden, gateway.Message))
			return
		}
		c.Next()
	}
}
