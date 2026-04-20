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

func Login(userEntity repository.IUser, sessionEntity repository.ISession, loginGuard repository.ILoginGuard) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.Login{}
		if err := ctx.ShouldBind(&req); err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		locked, err := loginGuard.IsLocked(req.Username)
		if err != nil {
			logrus.Warn("loginGuard IsLocked error: ", err)
		} else if locked {
			errs.Response(ctx, http.StatusTooManyRequests, errs.New(errs.ErrRateLimited, "account locked due to too many failed login attempts, try again later"))
			return
		}

		user, err := userEntity.GetUserByUsername(req.Username)
		if err != nil || user == nil {
			_ = loginGuard.RecordFailure(req.Username)
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrWrongCredentials, "wrong username or password"))
			return
		}
		if utils.ComparePasswordAndHashedPassword(req.Password, user.Password) != nil {
			_ = loginGuard.RecordFailure(req.Username)
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrWrongCredentials, "wrong username or password"))
			return
		}

		if user.Status != constant.ACTIVE {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrWrongCredentials, "wrong username or password"))
			return
		}

		_ = loginGuard.Reset(req.Username)

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
		if err := sessionEntity.RemoveSessionById(sessionId); err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		result := gin.H{
			"message": "success",
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func CreateSSOTicket(ssoEntity repository.ISSOTicket, userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userId := ctx.GetString(middlewares.UserId)
		user, err := userEntity.GetUserById(userId)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "user")
			return
		}
		if user.Status != constant.ACTIVE {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrTokenInvalid, "token invalid"))
			return
		}

		payload := repository.TicketPayload{
			UserId:   userId,
			Role:     ctx.GetString(middlewares.Role),
			ClientId: ctx.GetString(middlewares.ClientId),
			System:   ctx.GetString(middlewares.System),
		}
		ticket, err := ssoEntity.Create(payload)
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"ticket":    ticket,
			"expiresIn": int(repository.SSOTicketTTL.Seconds()),
		})
	}
}

func ExchangeSSOTicket(ssoEntity repository.ISSOTicket, userEntity repository.IUser, sessionEntity repository.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.ExchangeTicket{}
		if err := ctx.ShouldBind(&req); err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		payload, err := ssoEntity.Consume(req.Ticket)
		if err != nil {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrTokenInvalid, "ticket invalid or expired"))
			return
		}

		user, err := userEntity.GetUserById(payload.UserId)
		if err != nil || user == nil {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrTokenInvalid, "ticket invalid or expired"))
			return
		}
		if user.Status != constant.ACTIVE {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrTokenInvalid, "token invalid"))
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
			System:         payload.System,
			ClientId:       user.ClientId,
			ExpirationTime: expireDate,
		}
		token, err := middlewares.GenerateJwtToken(param)
		if err != nil {
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrTokenGenFailed, "failed to generate token"))
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"accessToken": token})
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
