package errs

import (
	"net/http"
	"strconv"
	"strings"

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

// Status returns the HTTP status encoded in the code (UM-403-002 → 403),
// or 500 when the code does not carry one.
func (e *AppError) Status() int {
	parts := strings.Split(e.Code, "-")
	if len(parts) == 3 {
		if status, err := strconv.Atoi(parts[1]); err == nil && http.StatusText(status) != "" {
			return status
		}
	}
	return http.StatusInternalServerError
}
