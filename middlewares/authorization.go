package middlewares

import (
	"net/http"
	"um/app/core/errs"

	"github.com/gin-gonic/gin"
)

func RequireAuthorization(auths ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role := ctx.GetString(Role)
		if role == "" {
			invalidRequest(ctx)
			return
		}
		isAccessible := false
		for _, auth := range auths {
			if role == auth {
				isAccessible = true
				break
			}
		}
		if !isAccessible {
			notPermission(ctx)
			return
		}
		ctx.Next()
	}
}

func invalidRequest(ctx *gin.Context) {
	errs.Response(ctx, http.StatusForbidden, errs.New(errs.ErrForbidden, "Invalid request, restricted endpoint"))
}

func notPermission(ctx *gin.Context) {
	errs.Response(ctx, http.StatusForbidden, errs.New(errs.ErrNoPermission, "Don't have permission"))
}
