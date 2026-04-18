package usecase

import (
	"net/http"
	"um/app/core/constant"
	"um/app/core/errs"
	"um/app/core/utils"
	"um/app/domain/repository"
	"um/app/featues/request"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func GetUserList(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role := ctx.GetString(middlewares.Role)
		var result interface{}
		var err error

		switch role {
		case constant.SUPER:
			result, err = userEntity.GetUsers()
		case constant.ADMIN:
			clientId := ctx.GetString(middlewares.ClientId)
			result, err = userEntity.GetUserAll(clientId)
		default:
			errs.Response(ctx, http.StatusForbidden, errs.New(errs.ErrNoPermission, "Don't have permission"))
			return
		}

		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func AddUserByRole(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.User{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		role := ctx.GetString(middlewares.Role)
		var targetRole string

		switch role {
		case constant.SUPER:
			if len(req.ClientId) != 3 {
				errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrInvalidClientId, "invalid client id"))
				return
			}
			targetRole = constant.ADMIN
		case constant.ADMIN:
			clientId := ctx.GetString(middlewares.ClientId)
			if len(req.ClientId) != 3 || req.ClientId != clientId {
				errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrInvalidClientId, "invalid client id"))
				return
			}
			targetRole = constant.USER
		default:
			errs.Response(ctx, http.StatusForbidden, errs.New(errs.ErrNoPermission, "Don't have permission"))
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		found, _ := userEntity.GetUserByUsername(req.Username)
		if found != nil {
			errs.Response(ctx, http.StatusConflict, errs.New(errs.ErrUsernameTaken, "username is taken"))
			return
		}

		req.CreatedBy = userId
		result, err := userEntity.CreateUser(req, targetRole)
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func ChangePassword(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.ChangePassword{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		user, err := userEntity.GetUserById(userId)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}

		if user == nil || utils.ComparePasswordAndHashedPassword(req.OldPassword, user.Password) != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrWrongPassword, "wrong password"))
			return
		}

		clientId := ctx.GetString(middlewares.ClientId)
		result, err := userEntity.ChangePassword(user.Id.Hex(), clientId, req)
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func DeleteUserById(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userId := ctx.GetString(middlewares.UserId)
		id := ctx.Param("id")
		if userId == id {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrDeleteSelf, "can't delete self user"))
			return
		}

		role := ctx.GetString(middlewares.Role)
		user, err := getAccessibleUser(ctx, userEntity, id)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}
		err = ValidateUserRole(role, user)
		if err != nil {
			errs.Response(ctx, http.StatusForbidden, errs.New(errs.ErrInvalidRolePermission, err.Error()))
			return
		}

		clientId := clientIdForRole(ctx)
		result, err := userEntity.RemoveUserById(id, clientId)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func GetUserById(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")

		clientId := clientIdForRole(ctx)
		result, err := userEntity.GetUserByClientId(id, clientId)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}
		ctx.JSON(http.StatusOK, result)
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

func SetPassword(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.SetPassword{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		id := ctx.Param("id")
		role := ctx.GetString(middlewares.Role)
		user, err := getAccessibleUser(ctx, userEntity, id)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}

		err = ValidateUserRole(role, user)
		if err != nil {
			errs.Response(ctx, http.StatusForbidden, errs.New(errs.ErrInvalidRolePermission, err.Error()))
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		req.UpdatedBy = userId
		clientId := clientIdForRole(ctx)
		result, err := userEntity.SetPassword(id, clientId, req)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}

		ctx.JSON(http.StatusOK, result)
	}
}

func UpdateRoleById(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.UpdateRole{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		if req.Role != constant.SUPER && req.Role != constant.ADMIN && req.Role != constant.USER {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrInvalidRole, "invalid role"))
			return
		}

		role := ctx.GetString(middlewares.Role)
		if req.Role == constant.SUPER && role != constant.SUPER {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrInvalidRole, "invalid role"))
			return
		}

		id := ctx.Param("id")
		user, err := getAccessibleUser(ctx, userEntity, id)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}

		err = ValidateUserRole(role, user)
		if err != nil {
			errs.Response(ctx, http.StatusForbidden, errs.New(errs.ErrInvalidRolePermission, err.Error()))
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		clientId := clientIdForRole(ctx)
		req.UpdatedBy = userId
		result, err := userEntity.UpdateRoleById(id, clientId, req)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func UpdateStatusById(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.UpdateStatus{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		if req.Status != constant.ACTIVE && req.Status != constant.INACTIVE {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, "invalid status"))
			return
		}

		role := ctx.GetString(middlewares.Role)

		id := ctx.Param("id")
		user, err := getAccessibleUser(ctx, userEntity, id)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}
		err = ValidateUserRole(role, user)
		if err != nil {
			errs.Response(ctx, http.StatusForbidden, errs.New(errs.ErrInvalidRolePermission, err.Error()))
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		clientId := clientIdForRole(ctx)
		req.UpdatedBy = userId
		result, err := userEntity.UpdateStatusById(id, clientId, req)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}
		ctx.JSON(http.StatusOK, result)
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

func UpdateUserById(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.UpdateUser{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		id := ctx.Param("id")
		userId := ctx.GetString(middlewares.UserId)
		if userId != id {
			role := ctx.GetString(middlewares.Role)
			user, err := getAccessibleUser(ctx, userEntity, id)
			if err != nil {
				logrus.Error(err)
				respondRepositoryError(ctx, err, "user")
				return
			}
			err = ValidateUserRole(role, user)
			if err != nil {
				errs.Response(ctx, http.StatusForbidden, errs.New(errs.ErrInvalidRolePermission, err.Error()))
				return
			}
		}

		clientId := clientIdForRole(ctx)

		req.UpdatedBy = userId
		result, err := userEntity.UpdateUserById(id, clientId, req)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}
