package api

import (
	"github.com/gin-gonic/gin"
	"um/app/domain/repository"
	"um/app/domain/usecase"
	"um/middlewares"
)

func ApplyUserAPI(
	app *gin.RouterGroup,
	userEntity repository.IUser,
	sessionEntity repository.ISession,
	systemEntity repository.ISystem,
) {

	route := app.Group("/user")

	route.GET("/info",
		middlewares.RequireAuthenticated(sessionEntity),
		usecase.GetUserInfo(userEntity),
	)

	route.PUT("/info",
		middlewares.RequireAuthenticated(sessionEntity),
		usecase.UpdateUserInfo(userEntity),
	)

	route.PUT("/change-password",
		middlewares.RequireAuthenticated(sessionEntity),
		usecase.ChangePassword(userEntity),
	)

	route.POST("/set-password",
		middlewares.RequireAuthenticated(sessionEntity),
		usecase.SetPassword(userEntity),
	)

	route.GET("/system",
		middlewares.RequireAuthenticated(sessionEntity),
		usecase.GetSystem(systemEntity),
	)
}
