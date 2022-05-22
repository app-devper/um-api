package api

import (
	"github.com/gin-gonic/gin"
	"um/app/domain/repository"
	"um/app/domain/usecase"
	"um/middlewares"
)

func ApplyAuthAPI(
	app *gin.RouterGroup,
	userEntity repository.IUser,
	sessionEntity repository.ISession,
) {

	route := app.Group("auth")

	route.POST("/login",
		usecase.Login(userEntity, sessionEntity),
	)

	route.GET("/keep-alive",
		middlewares.RequireAuthenticated(sessionEntity),
		usecase.KeepAlive(userEntity, sessionEntity),
	)

	route.POST("/verify-password",
		middlewares.RequireAuthenticated(sessionEntity),
		usecase.VerifyPassword(userEntity),
	)

	route.POST("/logout",
		middlewares.RequireAuthenticated(sessionEntity),
		usecase.Logout(sessionEntity),
	)
}
