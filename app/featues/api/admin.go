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

	route := app.Group("admin/:clientId/user")

	route.GET("",
		middlewares.RequireAuthenticated(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		middlewares.RequireClient(),
		usecase.GetUsersByClientId(userEntity),
	)

	route.POST("",
		middlewares.RequireAuthenticated(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		middlewares.RequireClient(),
		usecase.AddUser(userEntity),
	)

	route.GET("/:id",
		middlewares.RequireAuthenticated(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		middlewares.RequireClient(),
		usecase.GetUserById(userEntity),
	)

	route.DELETE("/:id",
		middlewares.RequireAuthenticated(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		middlewares.RequireClient(),
		usecase.DeleteUserById(userEntity),
	)

	route.PUT("/:id",
		middlewares.RequireAuthenticated(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		middlewares.RequireClient(),
		usecase.UpdateUserById(userEntity),
	)

	route.PATCH("/:id/status",
		middlewares.RequireAuthenticated(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		middlewares.RequireClient(),
		usecase.UpdateStatusById(userEntity),
	)

	route.PATCH("/:id/role",
		middlewares.RequireAuthenticated(sessionEntity),
		middlewares.RequireAuthorization(constant.ADMIN),
		middlewares.RequireClient(),
		usecase.UpdateRoleById(userEntity),
	)
}
