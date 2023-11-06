package api

import (
	"github.com/gin-gonic/gin"
	"um/app/core/constant"
	"um/app/domain/repository"
	"um/app/domain/usecase"
	"um/middlewares"
)

func ApplySuperUserAPI(
	app *gin.RouterGroup,
	userEntity repository.IUser,
	sessionEntity repository.ISession,
) {

	route := app.Group("super/user")

	route.GET("",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.RequireSession(sessionEntity),
		usecase.GetUsers(userEntity),
	)

	route.POST("",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.RequireSession(sessionEntity),
		usecase.AddAdmin(userEntity),
	)

	route.GET("/:id",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.RequireSession(sessionEntity),
		usecase.GetUserById(userEntity),
	)

	route.DELETE("/:id",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.RequireSession(sessionEntity),
		usecase.DeleteUserById(userEntity),
	)

	route.PUT("/:id",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.RequireSession(sessionEntity),
		usecase.UpdateUserById(userEntity),
	)

	route.PATCH("/:id/status",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.RequireSession(sessionEntity),
		usecase.UpdateStatusById(userEntity),
	)

	route.PATCH("/:id/role",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.RequireSession(sessionEntity),
		usecase.UpdateRoleById(userEntity),
	)
}
