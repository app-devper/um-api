package api

import (
	"github.com/gin-gonic/gin"
	"um/app/core/constant"
	"um/app/domain/repository"
	"um/app/domain/usecase"
	"um/middlewares"
)

func ApplyAdminUserAPI(
	app *gin.RouterGroup,
	userEntity repository.IUser,
	sessionEntity repository.ISession,
) {

	route := app.Group("admin/user")

	route.GET("",
		middlewares.RequireAuthenticated(),
		middlewares.RequireSession(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		usecase.GetUsersByClientId(userEntity),
	)

	route.POST("",
		middlewares.RequireAuthenticated(),
		middlewares.RequireSession(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		usecase.AddUser(userEntity),
	)

	route.GET("/:id",
		middlewares.RequireAuthenticated(),
		middlewares.RequireSession(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		usecase.GetUserById(userEntity),
	)

	route.DELETE("/:id",
		middlewares.RequireAuthenticated(),
		middlewares.RequireSession(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		usecase.DeleteUserById(userEntity),
	)

	route.PUT("/:id",
		middlewares.RequireAuthenticated(),
		middlewares.RequireSession(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		usecase.UpdateUserById(userEntity),
	)

	route.PATCH("/:id/status",
		middlewares.RequireAuthenticated(),
		middlewares.RequireSession(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		usecase.UpdateStatusById(userEntity),
	)

	route.PATCH("/:id/role",
		middlewares.RequireAuthenticated(),
		middlewares.RequireSession(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		usecase.UpdateRoleById(userEntity),
	)
}
