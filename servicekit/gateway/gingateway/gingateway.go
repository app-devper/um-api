// Package gingateway adapts gateway to gin.
package gingateway

import (
	"github.com/app-devper/um-api/servicekit/gateway"
	"github.com/gin-gonic/gin"
)

// Middleware passes requests the hosts allow; refuse writes the service's
// own 403 response and must abort.
func Middleware(h gateway.Hosts, refuse gin.HandlerFunc) gin.HandlerFunc {
	if !h.Enabled() {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		if !h.Allows(c.Request) {
			refuse(c)
			c.Abort()
			return
		}
		c.Next()
	}
}
