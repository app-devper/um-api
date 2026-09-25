package api

import (
	"time"
	"um/app/domain/repository"
	"um/app/domain/session"
	"um/app/domain/usecase"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// ApplyAuthAPI registers login and SSO exchange on the public group and the
// session-management routes on the protected group.
func ApplyAuthAPI(
	public *gin.RouterGroup,
	protected *gin.RouterGroup,
	sessions *session.Manager,
	userEntity repository.IUser,
	sessionEntity repository.ISession,
	systemEntity repository.ISystem,
	loginGuard repository.ILoginGuard,
	ssoEntity repository.ISSOTicket,
	rdb *redis.Client,
) {

	open := public.Group("auth")

	open.POST("/login",
		middlewares.RateLimiter(rdb, 5, 1*time.Minute),
		usecase.Login(sessions, userEntity, systemEntity, loginGuard),
	)

	open.POST("/exchange",
		middlewares.RateLimiter(rdb, 10, 1*time.Minute),
		usecase.ExchangeSSOTicket(sessions, ssoEntity, userEntity),
	)

	route := protected.Group("auth")

	route.POST("/sso-ticket", usecase.CreateSSOTicket(ssoEntity, userEntity))
	route.GET("/keep-alive", usecase.KeepAlive(sessions))
	route.GET("/verify", usecase.VerifySession())
	route.GET("/system", usecase.GetSystem(systemEntity))
	route.POST("/verify-password", usecase.VerifyPassword(userEntity))
	route.POST("/logout", usecase.Logout(sessionEntity))
	route.GET("/sessions", usecase.ListSessions(sessionEntity))
	route.DELETE("/sessions", usecase.RevokeOtherSessions(sessionEntity))
	route.DELETE("/sessions/:id", usecase.RevokeSession(sessionEntity))
}
