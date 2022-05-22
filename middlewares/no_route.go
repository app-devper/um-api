package middlewares

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func NoRoute() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Service Missing / Not found."})
	}
}
