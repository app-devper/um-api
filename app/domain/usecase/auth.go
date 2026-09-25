package usecase

import (
	"errors"
	"net/http"
	"strings"
	"um/app/core/constant"
	"um/app/core/errs"
	"um/app/core/utils"
	"um/app/domain/repository"
	"um/app/domain/session"
	"um/app/featues/request"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func RequireSession(sessions *session.Manager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := ctx.GetHeader("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrMissingAuthHeader, "missing authorization header"))
			return
		}
		principal, err := sessions.Verify(token)
		if err != nil {
			respondSessionError(ctx, err)
			return
		}
		ctx.Set(middlewares.Principal, principal)
		ctx.Set(middlewares.SessionId, principal.SessionId)
		ctx.Set(middlewares.UserId, principal.UserId)
		ctx.Set(middlewares.Role, principal.Role)
		ctx.Set(middlewares.System, principal.System)
		ctx.Set(middlewares.ClientId, principal.ClientId)
		logrus.Infof("SessionId: %s UserId: %s Role: %s System: %s ClientId: %s",
			principal.SessionId, principal.UserId, principal.Role, principal.System, principal.ClientId)
		ctx.Next()
	}
}

func Login(sessions *session.Manager, userEntity repository.IUser, systemEntity repository.ISystem, loginGuard repository.ILoginGuard) gin.HandlerFunc {
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

		token, err := sessions.Issue(user, req.System, sessionMetadata(ctx))
		if err != nil {
			respondSessionError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"accessToken": token})
	}
}

func KeepAlive(sessions *session.Manager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		principal, ok := ctx.Value(middlewares.Principal).(*session.Principal)
		if !ok {
			logrus.Error("KeepAlive: no session principal; route is missing RequireSession")
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		token, err := sessions.Renew(principal)
		if err != nil {
			respondSessionError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"accessToken": token})
	}
}

// VerifySession answers ADR-0001: another service forwards a user's token and
// gets back the live Session behind it. RequireSession has already done the
// verification; this only reports it and forbids caching by intermediaries.
func VerifySession() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		principal, ok := ctx.Value(middlewares.Principal).(*session.Principal)
		if !ok {
			logrus.Error("VerifySession: no session principal; route is missing RequireSession")
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		ctx.Header("Cache-Control", "no-store")
		ctx.JSON(http.StatusOK, gin.H{
			"sessionId": principal.SessionId,
			"userId":    principal.UserId,
			"role":      principal.Role,
			"system":    principal.System,
			"clientId":  principal.ClientId,
		})
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

func ExchangeSSOTicket(sessions *session.Manager, ssoEntity repository.ISSOTicket, userEntity repository.IUser) gin.HandlerFunc {
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

		token, err := sessions.Issue(user, payload.System, sessionMetadata(ctx))
		if err != nil {
			respondSessionError(ctx, err)
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

		target, err := sessionEntity.GetSessionById(targetId)
		if err != nil {
			errs.Response(ctx, http.StatusNotFound, errs.New(errs.ErrNotFound, "session not found"))
			return
		}
		if target.UserId != userId {
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

func sessionMetadata(ctx *gin.Context) session.Metadata {
	return session.Metadata{
		UserAgent: ctx.Request.UserAgent(),
		IPAddress: ctx.ClientIP(),
	}
}

func respondSessionError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, session.ErrTokenInvalid), errors.Is(err, session.ErrUserInactive):
		logrus.Warn(err)
		errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrTokenInvalid, "token invalid"))
	case errors.Is(err, session.ErrSessionInvalid):
		logrus.Warn(err)
		errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrSessionInvalid, "session invalid"))
	case errors.Is(err, session.ErrTokenGenFailed):
		logrus.Error(err)
		errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrTokenGenFailed, "failed to generate token"))
	default:
		logrus.Error(err)
		errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
	}
}
