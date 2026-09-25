package usecase

import (
	"errors"
	"net/http"
	"um/app/core/errs"
	"um/app/domain/repository"
	"um/app/domain/useradmin"
	"um/app/featues/request"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func GetUserList(admin *useradmin.Admin) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		result, err := admin.List(actorFrom(ctx))
		respondUserAdmin(ctx, result, err)
	}
}

func AddUserByRole(admin *useradmin.Admin) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.User{}
		if !bind(ctx, &req) {
			return
		}
		result, err := admin.Create(actorFrom(ctx), req)
		respondUserAdmin(ctx, result, err)
	}
}

func ChangePassword(admin *useradmin.Admin) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.ChangePassword{}
		if !bind(ctx, &req) {
			return
		}
		result, err := admin.ChangeOwnPassword(actorFrom(ctx), req)
		respondUserAdmin(ctx, result, err)
	}
}

func DeleteUserById(admin *useradmin.Admin) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		result, err := admin.Delete(actorFrom(ctx), ctx.Param("id"))
		respondUserAdmin(ctx, result, err)
	}
}

func GetUserById(admin *useradmin.Admin) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		result, err := admin.Get(actorFrom(ctx), ctx.Param("id"))
		respondUserAdmin(ctx, result, err)
	}
}

func GetUserInfo(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userId := ctx.GetString(middlewares.UserId)
		result, err := userEntity.GetUserById(userId)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func SetPassword(admin *useradmin.Admin) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.SetPassword{}
		if !bind(ctx, &req) {
			return
		}
		result, err := admin.SetPassword(actorFrom(ctx), ctx.Param("id"), req)
		respondUserAdmin(ctx, result, err)
	}
}

func UpdateRoleById(admin *useradmin.Admin) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.UpdateRole{}
		if !bind(ctx, &req) {
			return
		}
		result, err := admin.SetRole(actorFrom(ctx), ctx.Param("id"), req)
		respondUserAdmin(ctx, result, err)
	}
}

func UpdateStatusById(admin *useradmin.Admin) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.UpdateStatus{}
		if !bind(ctx, &req) {
			return
		}
		result, err := admin.SetStatus(actorFrom(ctx), ctx.Param("id"), req)
		respondUserAdmin(ctx, result, err)
	}
}

func UpdateUserInfo(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.UpdateUser{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		clientId := ctx.GetString(middlewares.ClientId)

		req.UpdatedBy = userId
		result, err := userEntity.UpdateUserById(userId, clientId, req)
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func UnlockUserById(admin *useradmin.Admin) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := admin.Unlock(actorFrom(ctx), ctx.Param("id")); err != nil {
			respondUserAdmin(ctx, nil, err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}

func UpdateUserById(admin *useradmin.Admin) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.UpdateUser{}
		if !bind(ctx, &req) {
			return
		}
		result, err := admin.Update(actorFrom(ctx), ctx.Param("id"), req)
		respondUserAdmin(ctx, result, err)
	}
}

func actorFrom(ctx *gin.Context) useradmin.Actor {
	return useradmin.Actor{
		UserId:    ctx.GetString(middlewares.UserId),
		SessionId: ctx.GetString(middlewares.SessionId),
		Role:      ctx.GetString(middlewares.Role),
		ClientId:  ctx.GetString(middlewares.ClientId),
	}
}

func bind(ctx *gin.Context, req any) bool {
	if err := ctx.ShouldBind(req); err != nil {
		errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
		return false
	}
	return true
}

func respondUserAdmin(ctx *gin.Context, result any, err error) {
	if err == nil {
		ctx.JSON(http.StatusOK, result)
		return
	}
	var denial *errs.AppError
	if errors.As(err, &denial) {
		errs.Response(ctx, denial.Status(), denial)
		return
	}
	logrus.Error(err)
	respondRepositoryError(ctx, err, "user")
}
