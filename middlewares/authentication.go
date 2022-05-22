package middlewares

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
	"time"
	"um/app/core/constant"
	"um/app/domain/repository"
)

type AccessClaims struct {
	UserRefId string `json:"userRefId"`
	Role      string `json:"role"`
	System    string `json:"system"`
	jwt.StandardClaims
}

func GenerateJwtToken(userRefId string, role string, system string, expirationTime time.Time) string {
	var jwtKey = []byte(os.Getenv("SECRET_KEY"))
	claims := &AccessClaims{
		UserRefId: userRefId,
		Role:      role,
		System:    system,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
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

func RequireAuthenticated(sessionEntity repository.ISession) gin.HandlerFunc {
	jwtKey := []byte(os.Getenv("SECRET_KEY"))
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		if token == "" {
			err := errors.New("missing authorization header")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		jwtToken := strings.Split(token, "Bearer ")
		if len(jwtToken) < 2 {
			err := errors.New("missing authorization header")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
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
		if tkn == nil || !tkn.Valid || claims.UserRefId == "" {
			err := errors.New("token invalid authorization header")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		session, err := sessionEntity.GetSessionById(claims.UserRefId)
		if session == nil {
			err := errors.New("user ref invalid")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if session.Objective != constant.AccessApi {
			err := errors.New("objective invalid")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if session.System != claims.System {
			err := errors.New("system invalid")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		ctx.Set("UserRefId", claims.UserRefId)
		ctx.Set("Role", claims.Role)
		ctx.Set("System", claims.System)
		ctx.Set("UserId", session.UserId.Hex())
		ctx.Set("ClientId", session.ClientId)

		logrus.Info("UserRefId: " + claims.UserRefId)
		logrus.Info("Role: " + claims.Role)
		logrus.Info("System: " + claims.System)
		logrus.Info("UserId: " + session.UserId.Hex())
		logrus.Info("ClientId: " + session.ClientId)
		return
	}
}
