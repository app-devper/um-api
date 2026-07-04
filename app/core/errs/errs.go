package errs

import (
	"github.com/gin-gonic/gin"
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func New(code string, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func (e *AppError) Error() string {
	return e.Message
}

func Response(ctx *gin.Context, httpStatus int, appErr *AppError) {
	ctx.AbortWithStatusJSON(httpStatus, appErr)
}
