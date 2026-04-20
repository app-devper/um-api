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
		usecase.Login(userEntity, sessionEntity, loginGuard),
	)

	route.POST("/sso-ticket",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity),
		usecase.CreateSSOTicket(ssoEntity, userEntity),
	)

	route.POST("/exchange",
		middlewares.RateLimiter(rdb, 10, 1*time.Minute),
		usecase.ExchangeSSOTicket(ssoEntity, userEntity, sessionEntity),
	)

	route.GET("/keep-alive",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity),
		usecase.KeepAlive(userEntity, sessionEntity),
	)

	route.GET("/system",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity),
		usecase.GetSystem(systemEntity),
	)

	route.POST("/verify-password",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity),
		usecase.VerifyPassword(userEntity),
	)

	route.POST("/logout",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity),
		usecase.Logout(sessionEntity),
	)

	route.GET("/sessions",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity),
		usecase.ListSessions(sessionEntity),
	)

	route.DELETE("/sessions",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity),
		usecase.RevokeOtherSessions(sessionEntity),
	)

	route.DELETE("/sessions/:id",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity),
		usecase.RevokeSession(sessionEntity),
	)
}
