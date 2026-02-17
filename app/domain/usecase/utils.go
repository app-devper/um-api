package usecase

import (
	"um/app/core/constant"
	"um/app/core/errs"
	"um/app/domain/model"

	"um/middlewares"

	"github.com/gin-gonic/gin"
)

// clientIdForRole returns the clientId from context for non-SUPER roles,
// or empty string for SUPER (allowing cross-client access).
func clientIdForRole(ctx *gin.Context) string {
	role := ctx.GetString(middlewares.Role)
	if role == constant.SUPER {
		return ""
	}
	return ctx.GetString(middlewares.ClientId)
}

func ValidateUserRole(role string, user *model.User) error {
	switch role {
	case constant.SUPER:
		if user.Role == constant.SUPER {
			return errs.New(errs.ErrInvalidRolePermission, "invalid role permission")
		}
	case constant.ADMIN:
		if user.Role == constant.SUPER || user.Role == constant.ADMIN {
			return errs.New(errs.ErrInvalidRolePermission, "invalid role permission")
		}
	default:
		return errs.New(errs.ErrInvalidRolePermission, "invalid role permission")
	}
	return nil
}
