package api

import (
	"um/app/core/constant"
	"um/app/domain/repository"
	"um/app/domain/usecase"
	"um/middlewares"

	"github.com/gin-gonic/gin"
)

func ApplyUserAPI(
	app *gin.RouterGroup,
	userEntity repository.IUser,
	sessionEntity repository.ISession,
	systemEntity repository.ISystem,
	loginGuard repository.ILoginGuard,
) {

	route := app.Group("/user")

	// Self-service (any authenticated user)
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

	// Management (SUPER + ADMIN)
	route.GET("",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.RequireSession(sessionEntity),
		usecase.GetUserList(userEntity),
	)

	route.POST("",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.RequireSession(sessionEntity),
		usecase.AddUserByRole(userEntity, systemEntity),
	)

	route.GET("/:id",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.RequireSession(sessionEntity),
		usecase.GetUserById(userEntity),
	)

	route.DELETE("/:id",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.RequireSession(sessionEntity),
		usecase.DeleteUserById(userEntity),
	)

	route.PUT("/:id",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.RequireSession(sessionEntity),
		usecase.UpdateUserById(userEntity),
	)

	route.PATCH("/:id/status",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.RequireSession(sessionEntity),
		usecase.UpdateStatusById(userEntity),
	)

	route.PATCH("/:id/role",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.RequireSession(sessionEntity),
		usecase.UpdateRoleById(userEntity),
	)

	route.PATCH("/:id/set-password",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.RequireSession(sessionEntity),
		usecase.SetPassword(userEntity),
	)

	route.POST("/:id/unlock",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.RequireSession(sessionEntity),
		usecase.UnlockUserById(userEntity, loginGuard),
	)
}
