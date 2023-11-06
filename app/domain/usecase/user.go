package usecase

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"um/app/core/constant"
	"um/app/core/utils"
	"um/app/domain/repository"
	"um/app/featues/request"
	"um/middlewares"
)

func GetUsers(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		result, err := userEntity.GetUsers()
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func AddAdmin(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.User{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		found, _ := userEntity.GetUserByUsername(req.Username)
		if found != nil {
			ctx.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "username is taken"})
			return
		}

		req.CreatedBy = userId
		result, err := userEntity.CreateUser(req, constant.ADMIN)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func AddUser(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.User{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		clientId := ctx.GetString(middlewares.ClientId)
		if len(req.ClientId) != 3 || req.ClientId != clientId {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		found, _ := userEntity.GetUserByUsername(req.Username)
		if found != nil {
			ctx.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "username is taken"})
			return
		}

		req.CreatedBy = userId
		result, err := userEntity.CreateUser(req, constant.USER)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		user, err := userEntity.GetUserById(userId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if (user == nil) || utils.ComparePasswordAndHashedPassword(req.OldPassword, user.Password) != nil {
			err = errors.New("wrong password")
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		clientId := ctx.GetString(middlewares.ClientId)
		result, err := userEntity.ChangePassword(user.Id.Hex(), clientId, req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "can't delete self user"})
			return
		}

		clientId := ctx.GetString(middlewares.ClientId)
		result, err := userEntity.RemoveUserById(id, clientId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func GetUserById(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		clientId := ctx.GetString(middlewares.ClientId)
		result, err := userEntity.GetUserByClientId(id, clientId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func GetUsersByClientId(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientId := ctx.GetString(middlewares.ClientId)
		result, err := userEntity.GetUserAll(clientId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		clientId := ctx.GetString(middlewares.ClientId)
		result, err := userEntity.SetPassword(userId, clientId, req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		role := ctx.GetString(middlewares.Role)
		if req.Role == constant.SUPER && role != constant.SUPER {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
			return
		}

		id := ctx.Param("id")
		err = userEntity.ValidateUserRole(role, id)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		clientId := ctx.GetString(middlewares.ClientId)
		req.UpdatedBy = userId
		result, err := userEntity.UpdateRoleById(id, clientId, req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		role := ctx.GetString(middlewares.Role)

		id := ctx.Param("id")
		err = userEntity.ValidateUserRole(role, id)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		clientId := ctx.GetString(middlewares.ClientId)
		req.UpdatedBy = userId
		result, err := userEntity.UpdateStatusById(id, clientId, req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		clientId := ctx.GetString(middlewares.ClientId)

		req.UpdatedBy = userId
		result, err := userEntity.UpdateUserById(userId, clientId, req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		id := ctx.Param("id")

		userId := ctx.GetString(middlewares.UserId)
		if userId != id {
			role := ctx.GetString(middlewares.Role)
			err = userEntity.ValidateUserRole(role, id)
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
				return
			}
		}

		clientId := ctx.GetString(middlewares.ClientId)

		req.UpdatedBy = userId
		result, err := userEntity.UpdateUserById(id, clientId, req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}
