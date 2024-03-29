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
) {

	route := app.Group("/user")

	route.GET("/info",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity),
		usecase.GetUserInfo(userEntity),
	)

	route.PUT("/info",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity),
		usecase.UpdateUserInfo(userEntity),
	)

	route.PUT("/change-password",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity),
		usecase.ChangePassword(userEntity),
	)

	route.POST("/set-password",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity),
		usecase.SetPassword(userEntity),
	)
}
