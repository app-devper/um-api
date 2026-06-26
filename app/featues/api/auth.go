package api

import (
	"time"
	"um/app/domain/repository"
	"um/app/domain/usecase"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func ApplyAuthAPI(
	app *gin.RouterGroup,
	secretKey string,
	userEntity repository.IUser,
	sessionEntity repository.ISession,
	systemEntity repository.ISystem,
	loginGuard repository.ILoginGuard,
	ssoEntity repository.ISSOTicket,
	rdb *redis.Client,
) {

	route := app.Group("auth")

	route.POST("/login",
		middlewares.RateLimiter(rdb, 5, 1*time.Minute),
		usecase.Login(secretKey, userEntity, sessionEntity, systemEntity, loginGuard),
	)

	route.POST("/sso-ticket",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.CreateSSOTicket(ssoEntity, userEntity),
	)

	route.POST("/exchange",
		middlewares.RateLimiter(rdb, 10, 1*time.Minute),
		usecase.ExchangeSSOTicket(secretKey, ssoEntity, userEntity, sessionEntity),
	)

	route.GET("/keep-alive",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.KeepAlive(secretKey, userEntity, sessionEntity),
	)

	route.GET("/system",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.GetSystem(systemEntity),
	)

	route.POST("/verify-password",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.VerifyPassword(userEntity),
	)

	route.POST("/logout",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.Logout(sessionEntity),
	)

	route.GET("/sessions",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.ListSessions(sessionEntity),
	)

	route.DELETE("/sessions",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.RevokeOtherSessions(sessionEntity),
	)

	route.DELETE("/sessions/:id",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.RevokeSession(sessionEntity),
	)
}
