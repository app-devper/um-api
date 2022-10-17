package middlewares

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func NewRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(recoveryHandler)
}

func recoveryHandler(ctx *gin.Context, err interface{}) {
	ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"error": err,
	})
}
