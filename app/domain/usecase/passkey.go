package usecase

import (
	"errors"
	"net/http"
	"um/app/core/errs"
	"um/app/domain/passkey"
	"um/app/featues/request"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PasskeyLoginOptions starts a username-less Passkey login on a System.
func PasskeyLoginOptions(passkeys *passkey.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.PasskeyLoginOptions{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}
		options, err := passkeys.BeginLogin(req.System)
		if err != nil {
			respondPasskeyError(ctx, err)
			return
		}
		ctx.Header("Cache-Control", "no-store")
		ctx.JSON(http.StatusOK, options)
	}
}

// PasskeyLogin verifies the assertion and starts a Session (ADR-0010).
func PasskeyLogin(passkeys *passkey.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.PasskeyLogin{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}
		token, err := passkeys.FinishLogin(req.CeremonyId, req.Credential, sessionMetadata(ctx))
		if err != nil {
			respondPasskeyError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"accessToken": token})
	}
}

// PasskeyRegistrationOptions starts adding a Passkey; the current password is required.
func PasskeyRegistrationOptions(passkeys *passkey.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.PasskeyRegistrationOptions{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}
		options, err := passkeys.BeginRegistration(ctx.GetString(middlewares.UserId), req.Password)
		if err != nil {
			respondPasskeyError(ctx, err)
			return
		}
		ctx.Header("Cache-Control", "no-store")
		ctx.JSON(http.StatusOK, options)
	}
}

// RegisterPasskey verifies the new credential and stores it.
func RegisterPasskey(passkeys *passkey.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.PasskeyRegistration{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}
		key, err := passkeys.FinishRegistration(ctx.GetString(middlewares.UserId), req.CeremonyId, req.Name, req.Credential)
		if err != nil {
			respondPasskeyError(ctx, err)
			return
		}
		ctx.JSON(http.StatusCreated, key)
	}
}

// ListPasskeys lists the caller's own Passkeys.
func ListPasskeys(passkeys *passkey.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		keys, err := passkeys.List(ctx.GetString(middlewares.UserId))
		if err != nil {
			respondPasskeyError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, keys)
	}
}

// DeletePasskey deletes one of the caller's own Passkeys.
func DeletePasskey(passkeys *passkey.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := passkeys.Delete(ctx.GetString(middlewares.UserId), ctx.Param("id")); err != nil {
			respondPasskeyError(ctx, err)
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

func respondPasskeyError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, passkey.ErrDisabled):
		errs.Response(ctx, http.StatusNotFound, errs.New(errs.ErrPasskeysDisabled, err.Error()))
	case errors.Is(err, passkey.ErrWrongPassword):
		errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrWrongPassword, err.Error()))
	case errors.Is(err, passkey.ErrCeremony):
		errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrCeremonyExpired, passkey.ErrCeremony.Error()))
	case errors.Is(err, passkey.ErrNotAccepted):
		logrus.Warn(err)
		errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrPasskeyRejected, passkey.ErrNotAccepted.Error()))
	case errors.Is(err, passkey.ErrNotFound):
		errs.Response(ctx, http.StatusNotFound, errs.New(errs.ErrNotFound, err.Error()))
	default:
		respondSessionError(ctx, err)
	}
}
