package usecase

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"um/app/core/constant"
	"um/app/core/utils"
	"um/app/domain/repository"
	"um/app/featues/request"
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

		userId := ctx.GetString("UserId")
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

		clientId := ctx.GetString("ClientId")
		if len(req.ClientId) != 3 || req.ClientId != clientId {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
			return
		}

		userId := ctx.GetString("UserId")
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

		userId := ctx.GetString("UserId")
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

		clientId := ctx.GetString("ClientId")
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
		userId := ctx.GetString("UserId")
		id := ctx.Param("id")
		if userId == id {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "can't delete self user"})
			return
		}

		clientId := ctx.GetString("ClientId")
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
		clientId := ctx.GetString("ClientId")
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
		userId := ctx.GetString("UserId")
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
		clientId := ctx.GetString("ClientId")
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

		userId := ctx.GetString("UserId")
		clientId := ctx.GetString("ClientId")
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

		userId := ctx.GetString("UserId")
		id := ctx.Param("id")
		role := ctx.GetString("Role")
		clientId := ctx.GetString("ClientId")

		if req.Role == constant.SUPER && role != constant.SUPER {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
			return
		}

		err = userEntity.ValidateUserRole(role, id)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

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

		id := ctx.Param("id")
		userId := ctx.GetString("UserId")
		role := ctx.GetString("Role")
		clientId := ctx.GetString("ClientId")

		err = userEntity.ValidateUserRole(role, id)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

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

		userId := ctx.GetString("UserId")
		clientId := ctx.GetString("ClientId")
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

		clientId := ctx.GetString("ClientId")
		userId := ctx.GetString("UserId")
		role := ctx.GetString("Role")
		id := ctx.Param("id")

		err = userEntity.ValidateUserRole(role, id)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		req.UpdatedBy = userId
		result, err := userEntity.UpdateUserById(id, clientId, req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}
