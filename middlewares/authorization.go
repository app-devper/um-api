package middlewares

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func RequireAuthorization(auths ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var roles []string
		roles = append(roles, ctx.GetString(Role))
		if len(roles) <= 0 {
			invalidRequest(ctx)
			return
		}
		isAccessible := false
		if len(roles) < len(auths) || len(roles) == len(auths) {
			for _, auth := range auths {
				for _, role := range roles {
					if role == auth {
						isAccessible = true
						break
					}
				}
			}
		}
		if len(roles) > len(auths) {
			for _, role := range roles {
				for _, auth := range auths {
					if auth == role {
						isAccessible = true
						break
					}
				}
			}
		}
		if isAccessible == false {
			notPermission(ctx)
			return
		}
		ctx.Next()
	}
}

func invalidRequest(ctx *gin.Context) {
	ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid request, restricted endpoint"})
}

func notPermission(ctx *gin.Context) {
	ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Don't have permission"})
}
