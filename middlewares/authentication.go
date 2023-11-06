package middlewares

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
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
	jwt.StandardClaims
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
		StandardClaims: jwt.StandardClaims{
			Id:        param.SessionId,
			ExpiresAt: param.ExpirationTime.Unix(),
			Audience:  "domain",
			Issuer:    "uit",
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
		if tkn == nil || !tkn.Valid || claims.Id == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalid"})
			return
		}

		ctx.Set(SessionId, claims.Id)
		ctx.Set(Role, claims.Role)
		ctx.Set(System, claims.System)
		ctx.Set(ClientId, claims.ClientId)

		logrus.Info("SessionId: " + claims.Id)
		logrus.Info("Role: " + claims.Role)
		logrus.Info("System: " + claims.System)
		logrus.Info("ClientId: " + claims.ClientId)
		return
	}
}
