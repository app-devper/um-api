package middlewares

import (
	"net/http"
	"um/app/core/errs"

	"github.com/gin-gonic/gin"
)

func NoRoute() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		errs.Response(ctx, http.StatusNotFound, errs.New(errs.ErrNotFound, "Service Missing / Not found."))
	}
}
