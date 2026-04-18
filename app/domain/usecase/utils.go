package usecase

import (
	"errors"
	"net/http"
	"um/app/core/constant"
	"um/app/core/errs"
	"um/app/domain/model"
	"um/app/domain/repository"

	"um/middlewares"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

func getAccessibleUser(ctx *gin.Context, userEntity repository.IUser, id string) (*model.User, error) {
	return userEntity.GetUserByClientId(id, clientIdForRole(ctx))
}

func respondRepositoryError(ctx *gin.Context, err error, resource string) {
	switch {
	case errors.Is(err, primitive.ErrInvalidHex):
		errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, "invalid id"))
	case errors.Is(err, mongo.ErrNoDocuments):
		errs.Response(ctx, http.StatusNotFound, errs.New(errs.ErrNotFound, resource+" not found"))
	default:
		errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
	}
}
