package usecase

import (
	"errors"
	"net/http"
	"strings"
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

// Login starts a Session from credentials; the session module owns every
// rule (ADR-0009).
func Login(sessions *session.Manager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.Login{}
		if err := ctx.ShouldBind(&req); err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}
		token, err := sessions.Start(session.Credentials{Username: req.Username, Password: req.Password, System: req.System}, sessionMetadata(ctx))
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

// Logout ends the current Session.
func Logout(sessions *session.Manager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := sessions.End(ctx.GetString(middlewares.SessionId)); err != nil {
			respondSessionError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}

// CreateSSOTicket hands the verified Session's User and System to another app
// of the same System; RequireSession has already checked the User is ACTIVE.
func CreateSSOTicket(ssoEntity repository.ISSOTicket) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload := repository.TicketPayload{
			UserId:   ctx.GetString(middlewares.UserId),
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

// ExchangeSSOTicket starts a Session from a one-time ticket on the ticket's
// System; the session module re-checks the User (ADR-0009).
func ExchangeSSOTicket(sessions *session.Manager, ssoEntity repository.ISSOTicket) gin.HandlerFunc {
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
		token, err := sessions.Resume(payload.UserId, payload.System, sessionMetadata(ctx))
		if err != nil {
			respondSessionError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"accessToken": token})
	}
}

func ListSessions(sessions *session.Manager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		items, err := sessions.List(ctx.GetString(middlewares.UserId), ctx.GetString(middlewares.SessionId))
		if err != nil {
			respondSessionError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, items)
	}
}

func RevokeSession(sessions *session.Manager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		err := sessions.EndOther(ctx.GetString(middlewares.UserId), ctx.GetString(middlewares.SessionId), ctx.Param("id"))
		if err != nil {
			respondSessionError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}

func RevokeOtherSessions(sessions *session.Manager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		count, err := sessions.EndAll(ctx.GetString(middlewares.UserId), ctx.GetString(middlewares.SessionId))
		if err != nil {
			respondSessionError(ctx, err)
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
	case errors.Is(err, session.ErrWrongCredentials), errors.Is(err, session.ErrNotActive), errors.Is(err, session.ErrWrongSystem):
		errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrWrongCredentials, err.Error()))
	case errors.Is(err, session.ErrLocked):
		errs.Response(ctx, http.StatusTooManyRequests, errs.New(errs.ErrRateLimited, err.Error()))
	case errors.Is(err, session.ErrUnavailable):
		logrus.Error(err)
		errs.Response(ctx, http.StatusServiceUnavailable, errs.New(errs.ErrUnavailable, "identity store unavailable, try again later"))
	case errors.Is(err, session.ErrNotEnded):
		logrus.Error(err)
		errs.Response(ctx, http.StatusServiceUnavailable, errs.New(errs.ErrSessionsNotEnded, "sessions could not be ended, try again"))
	case errors.Is(err, session.ErrCurrent):
		errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
	case errors.Is(err, session.ErrNotFound):
		errs.Response(ctx, http.StatusNotFound, errs.New(errs.ErrNotFound, err.Error()))
	case errors.Is(err, session.ErrNotOwner):
		errs.Response(ctx, http.StatusForbidden, errs.New(errs.ErrNoPermission, "Don't have permission"))
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
