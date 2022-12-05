package usecase

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
	"um/app/core/utils"
	"um/app/domain/repository"
	"um/app/featues/request"
	"um/config"
	"um/middlewares"
)

func Login(userEntity repository.IUser, sessionEntity repository.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.Login{}
		if err := ctx.ShouldBind(&req); err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		user, err := userEntity.GetUserByUsername(req.Username)
		if (user == nil) || utils.ComparePasswordAndHashedPassword(req.Password, user.Password) != nil {
			err = errors.New("wrong username or password")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		var expireDate = time.Now().Add(config.AccessTokenTime)

		sessionId, err := sessionEntity.CreateSession(user.Id.Hex(), config.AccessTokenTime)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		param := &middlewares.TokenParam{
			SessionId:      sessionId,
			Role:           user.Role,
			System:         req.System,
			ClientId:       user.ClientId,
			ExpirationTime: expireDate,
		}
		token := middlewares.GenerateJwtToken(param)
		result := gin.H{
			"accessToken": token,
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func KeepAlive(userEntity repository.IUser, sessionEntity repository.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sessionId := ctx.GetString("SessionId")
		userId := ctx.GetString("UserId")

		user, err := userEntity.GetUserById(userId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var expireDate = time.Now().Add(config.AccessTokenTime)
		err = sessionEntity.UpdateSessionExpireById(sessionId, config.AccessTokenTime)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		system := ctx.GetString("System")
		param := &middlewares.TokenParam{
			SessionId:      sessionId,
			Role:           user.Role,
			System:         system,
			ClientId:       user.ClientId,
			ExpirationTime: expireDate,
		}
		token := middlewares.GenerateJwtToken(param)
		result := gin.H{
			"accessToken": token,
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func Logout(sessionEntity repository.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sessionId := ctx.GetString("SessionId")
		_ = sessionEntity.RemoveSessionById(sessionId)
		result := gin.H{
			"message": "success",
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func VerifyPassword(userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.VerifyPassword{}
		if err := ctx.ShouldBind(&req); err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userId := ctx.GetString("UserId")
		user, err := userEntity.GetUserById(userId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if (user == nil) || utils.ComparePasswordAndHashedPassword(req.Password, user.Password) != nil {
			err = errors.New("wrong password")
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result := gin.H{
			"message": "success",
		}
		ctx.JSON(http.StatusOK, result)
	}
}
