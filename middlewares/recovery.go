package middlewares

import (
	"net/http"
	"um/app/core/errs"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func NewRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(recoveryHandler)
}

func recoveryHandler(ctx *gin.Context, err interface{}) {
	logrus.Errorf("panic recovered: %v", err)
	errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
}
