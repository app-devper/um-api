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
	rdb *redis.Client,
) {

	route := app.Group("auth")

	route.POST("/login",
		middlewares.RateLimiter(rdb, 5, 1*time.Minute),
		usecase.Login(userEntity, sessionEntity),
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
}
