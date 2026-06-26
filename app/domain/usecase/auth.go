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

func RequireSession(sessionEntity repository.ISession, userEntity repository.IUser) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sessionId := ctx.GetString(middlewares.SessionId)
		userId, err := sessionEntity.GetSessionById(sessionId)
		if err != nil {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrSessionInvalid, "session invalid"))
			return
		}
		user, err := userEntity.GetUserById(userId)
		if err != nil || user == nil {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrSessionInvalid, "session invalid"))
			return
		}
		if user.Status != constant.ACTIVE {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrTokenInvalid, "token invalid"))
			return
		}
		ctx.Set(middlewares.UserId, userId)
		ctx.Set(middlewares.Role, user.Role)
		ctx.Set(middlewares.ClientId, user.ClientId)
		logrus.Info("UserId: " + userId)
		ctx.Next()
	}
}

func Login(secretKey string, userEntity repository.IUser, sessionEntity repository.ISession, systemEntity repository.ISystem, loginGuard repository.ILoginGuard) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.Login{}
		if err := ctx.ShouldBind(&req); err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		username := utils.NormalizeUsername(req.Username)
		locked, err := loginGuard.IsLocked(username)
		if err != nil {
			logrus.Warn("loginGuard IsLocked error: ", err)
		} else if locked {
			errs.Response(ctx, http.StatusTooManyRequests, errs.New(errs.ErrRateLimited, "account locked due to too many failed login attempts, try again later"))
			return
		}

		user, err := userEntity.GetUserByUsername(username)
		if err != nil || user == nil {
			_ = loginGuard.RecordFailure(username)
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrWrongCredentials, "wrong username or password"))
			return
		}
		if utils.ComparePasswordAndHashedPassword(req.Password, user.Password) != nil {
			_ = loginGuard.RecordFailure(username)
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrWrongCredentials, "wrong username or password"))
			return
		}

		if user.Status != constant.ACTIVE {
			_ = loginGuard.RecordFailure(username)
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrWrongCredentials, "account is not active"))
			return
		}

		system, err := systemEntity.GetSystem(user.ClientId, req.System)
		if err != nil || system == nil {
			_ = loginGuard.RecordFailure(username)
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrWrongCredentials, "invalid client or system"))
			return
		}

		_ = loginGuard.Reset(username)

		expireDate := time.Now().Add(config.AccessTokenTime)

		sessionId, err := sessionEntity.CreateSession(user.Id.Hex(), config.AccessTokenTime, repository.SessionMetadata{
			UserAgent: ctx.Request.UserAgent(),
			IPAddress: ctx.ClientIP(),
			System:    req.System,
		})
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
		token, err := middlewares.GenerateJwtToken(secretKey, param)
		if err != nil {
			if removeErr := sessionEntity.RemoveSessionById(sessionId); removeErr != nil {
				logrus.Error(removeErr)
			}
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrTokenGenFailed, "failed to generate token"))
			return
		}
		result := gin.H{
			"accessToken": token,
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func KeepAlive(secretKey string, userEntity repository.IUser, sessionEntity repository.ISession) gin.HandlerFunc {
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
		token, err := middlewares.GenerateJwtToken(secretKey, param)
		if err != nil {
			if removeErr := sessionEntity.RemoveSessionById(sessionId); removeErr != nil {
				logrus.Error(removeErr)
			}
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

func ExchangeSSOTicket(secretKey string, ssoEntity repository.ISSOTicket, userEntity repository.IUser, sessionEntity repository.ISession) gin.HandlerFunc {
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
		sessionId, err := sessionEntity.CreateSession(user.Id.Hex(), config.AccessTokenTime, repository.SessionMetadata{
			UserAgent: ctx.Request.UserAgent(),
			IPAddress: ctx.ClientIP(),
			System:    payload.System,
		})
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
		token, err := middlewares.GenerateJwtToken(secretKey, param)
		if err != nil {
			if removeErr := sessionEntity.RemoveSessionById(sessionId); removeErr != nil {
				logrus.Error(removeErr)
			}
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrTokenGenFailed, "failed to generate token"))
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"accessToken": token})
	}
}

func ListSessions(sessionEntity repository.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userId := ctx.GetString(middlewares.UserId)
		currentSessionId := ctx.GetString(middlewares.SessionId)
		items, err := sessionEntity.ListUserSessions(userId, currentSessionId)
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		ctx.JSON(http.StatusOK, items)
	}
}

func RevokeSession(sessionEntity repository.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		targetId := ctx.Param("id")
		userId := ctx.GetString(middlewares.UserId)
		currentSessionId := ctx.GetString(middlewares.SessionId)

		if targetId == currentSessionId {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, "cannot revoke current session, use logout instead"))
			return
		}

		targetUserId, err := sessionEntity.GetSessionById(targetId)
		if err != nil {
			errs.Response(ctx, http.StatusNotFound, errs.New(errs.ErrNotFound, "session not found"))
			return
		}
		if targetUserId != userId {
			errs.Response(ctx, http.StatusForbidden, errs.New(errs.ErrNoPermission, "Don't have permission"))
			return
		}

		if err := sessionEntity.RemoveSessionById(targetId); err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}

func RevokeOtherSessions(sessionEntity repository.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userId := ctx.GetString(middlewares.UserId)
		currentSessionId := ctx.GetString(middlewares.SessionId)
		count, err := sessionEntity.RevokeOtherSessions(userId, currentSessionId)
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"message": "success", "revoked": count})
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
