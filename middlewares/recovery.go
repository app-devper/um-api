package middlewares

import (
	"github.com/ekyoung/gin-nice-recovery"
	"github.com/gin-gonic/gin"
	"net/http"
)

func NewRecovery() gin.HandlerFunc {
	return nice.Recovery(recoveryHandler)
}

func recoveryHandler(ctx *gin.Context, err interface{}) {
	ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"error": err,
	})
}
