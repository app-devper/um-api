package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
	"time"
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

func GenerateJwtToken(param *TokenParam) string {
	var jwtKey = []byte(os.Getenv("SECRET_KEY"))
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
		logrus.Error(err)
	}
	return tokenString
}

func RequireAuthenticated() gin.HandlerFunc {
	jwtKey := []byte(os.Getenv("SECRET_KEY"))
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		if token == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		jwtToken := strings.Split(token, "Bearer ")
		if len(jwtToken) < 2 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		claims := &AccessClaims{}
		tkn, err := jwt.ParseWithClaims(jwtToken[1], claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if tkn == nil || !tkn.Valid || claims.ID == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalid"})
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
		return
	}
}
