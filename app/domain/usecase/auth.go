package usecase

import (
	"net/http"
	"time"
	"um/app/core/config"
	"um/app/core/constant"
	"um/app/core/errs"
	"um/app/core/utils"
	"um/app/domain/repository"
	"um/app/featues/request"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func RequireSession(sessionEntity repository.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sessionId := ctx.GetString(middlewares.SessionId)
		userId, err := sessionEntity.GetSessionById(sessionId)
		if err != nil {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrSessionInvalid, "session invalid"))
			return
		}
		ctx.Set(middlewares.UserId, userId)
		logrus.Info("UserId: " + userId)
		ctx.Next()
	}
}

func Login(userEntity repository.IUser, sessionEntity repository.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.Login{}
		if err := ctx.ShouldBind(&req); err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}
		user, err := userEntity.GetUserByUsername(req.Username)
		if err != nil || user == nil {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrWrongCredentials, "wrong username or password"))
			return
		}
		if utils.ComparePasswordAndHashedPassword(req.Password, user.Password) != nil {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrWrongCredentials, "wrong username or password"))
			return
		}

		if user.Status != constant.ACTIVE {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrWrongCredentials, "wrong username or password"))
			return
		}

		expireDate := time.Now().Add(config.AccessTokenTime)

		sessionId, err := sessionEntity.CreateSession(user.Id.Hex(), config.AccessTokenTime)
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}

		param := &middlewares.TokenParam{
			SessionId:      sessionId,
			Role:           user.Role,
			System:         req.System,
			ClientId:       user.ClientId,
			ExpirationTime: expireDate,
		}
		token, err := middlewares.GenerateJwtToken(param)
		if err != nil {
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrTokenGenFailed, "failed to generate token"))
			return
		}
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
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}

		if user.Status != constant.ACTIVE {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrTokenInvalid, "token invalid"))
			return
		}

		expireDate := time.Now().Add(config.AccessTokenTime)
		err = sessionEntity.UpdateSessionExpireById(sessionId, config.AccessTokenTime)
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
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
		token, err := middlewares.GenerateJwtToken(param)
		if err != nil {
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrTokenGenFailed, "failed to generate token"))
			return
		}
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
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		userId := ctx.GetString(middlewares.UserId)
		user, err := userEntity.GetUserById(userId)
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}

		if user == nil || utils.ComparePasswordAndHashedPassword(req.Password, user.Password) != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrWrongPassword, "wrong password"))
			return
		}
		result := gin.H{
			"message": "success",
		}
		ctx.JSON(http.StatusOK, result)
	}
}
