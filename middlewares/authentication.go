package middlewares

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
	"um/app/core/errs"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

type AccessClaims struct {
	Role     string `json:"role"`
	System   string `json:"system"`
	ClientId string `json:"clientId"`
	jwt.RegisteredClaims
}

type TokenParam struct {
	SessionId      string
	Role           string
	System         string
	ClientId       string
	ExpirationTime time.Time
}

func GenerateJwtToken(param *TokenParam) (string, error) {
	key := os.Getenv("SECRET_KEY")
	if key == "" {
		return "", errors.New("SECRET_KEY is not set")
	}
	jwtKey := []byte(key)
	claims := &AccessClaims{
		Role:     param.Role,
		System:   param.System,
		ClientId: param.ClientId,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        param.SessionId,
			ExpiresAt: jwt.NewNumericDate(param.ExpirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func RequireAuthenticated() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := os.Getenv("SECRET_KEY")
		if key == "" {
			logrus.Error("SECRET_KEY is not set")
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		jwtKey := []byte(key)
		token := ctx.GetHeader("Authorization")
		if token == "" {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrMissingAuthHeader, "missing authorization header"))
			return
		}
		jwtToken := strings.Split(token, "Bearer ")
		if len(jwtToken) < 2 {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrMissingAuthHeader, "missing authorization header"))
			return
		}
		claims := &AccessClaims{}
		tkn, err := jwt.ParseWithClaims(jwtToken[1], claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil {
			logrus.Warn("token parse error: ", err)
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrTokenInvalid, "token invalid"))
			return
		}
		if tkn == nil || !tkn.Valid || claims.ID == "" {
			errs.Response(ctx, http.StatusUnauthorized, errs.New(errs.ErrTokenInvalid, "token invalid"))
			return
		}

		ctx.Set(SessionId, claims.ID)
		ctx.Set(Role, claims.Role)
		ctx.Set(System, claims.System)
		ctx.Set(ClientId, claims.ClientId)

		logrus.Info("SessionId: " + claims.ID)
		logrus.Info("Role: " + claims.Role)
		logrus.Info("System: " + claims.System)
		logrus.Info("ClientId: " + claims.ClientId)

		ctx.Next()
	}
}
