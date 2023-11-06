package usecase

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
	"um/app/core/config"
	"um/app/core/utils"
	"um/app/domain/repository"
	"um/app/featues/request"
	"um/middlewares"
)

func RequireSession(sessionEntity repository.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sessionId := ctx.GetString(middlewares.SessionId)
		userId, err := sessionEntity.GetSessionById(sessionId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session invalid"})
			return
		}
		ctx.Set(middlewares.UserId, userId)
		logrus.Info("UserId: " + userId)
		return
	}
}

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

		expireDate := time.Now().Add(config.AccessTokenTime)

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
		sessionId := ctx.GetString(middlewares.SessionId)
		userId := ctx.GetString(middlewares.UserId)

		user, err := userEntity.GetUserById(userId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		expireDate := time.Now().Add(config.AccessTokenTime)
		err = sessionEntity.UpdateSessionExpireById(sessionId, config.AccessTokenTime)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		system := ctx.GetString(middlewares.System)
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
		sessionId := ctx.GetString(middlewares.SessionId)
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

		userId := ctx.GetString(middlewares.UserId)
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
