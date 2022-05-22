package usecase

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
	"um/app/core/constant"
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
		data := request.Session{
			UserId:     user.Id,
			Type:       constant.AccessToken,
			Objective:  constant.AccessApi,
			System:     req.System,
			ClientId:   user.ClientId,
			ExpireDate: expireDate,
		}
		session, err := sessionEntity.CreateSession(data)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		token := middlewares.GenerateJwtToken(session.Id.Hex(), user.Role, session.System, expireDate)
		result := gin.H{
			"accessToken": token,
			"clientId":    session.ClientId,
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func KeepAlive(userEntity repository.IUser, sessionEntity repository.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userId := ctx.GetString("UserId")
		userRefId := ctx.GetString("UserRefId")
		user, err := userEntity.GetUserById(userId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		session, err := sessionEntity.GetSessionById(userRefId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var expireDate = time.Now().Add(config.AccessTokenTime)
		session, err = sessionEntity.UpdateSessionExpireById(userRefId, expireDate)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		token := middlewares.GenerateJwtToken(session.Id.Hex(), user.Role, session.System, expireDate)
		result := gin.H{
			"accessToken": token,
			"clientId":    session.ClientId,
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func Logout(sessionEntity repository.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userRefId := ctx.GetString("UserRefId")
		_, _ = sessionEntity.RemoveSessionById(userRefId)
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
