package middlewares

import (
	"net/http"

	"um/app/core/errs"

	"github.com/app-devper/um-api/servicekit/gateway"
	"github.com/app-devper/um-api/servicekit/gateway/gingateway"
	"github.com/gin-gonic/gin"
)

// NewGatewayHost refuses requests that did not come through the gateway
// (servicekit/gateway, ADR-0007), in UM's error envelope.
func NewGatewayHost(allowedHosts string) gin.HandlerFunc {
	return gingateway.Middleware(gateway.ParseHosts(allowedHosts), func(c *gin.Context) {
		errs.Response(c, http.StatusForbidden, errs.New(errs.ErrForbidden, gateway.Message))
	})
}
