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
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.GetUserInfo(userEntity),
	)

	route.PUT("/info",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.UpdateUserInfo(userEntity),
	)

	route.PUT("/change-password",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.ChangePassword(userEntity, sessionEntity),
	)

	// Management (SUPER + ADMIN)
	route.GET("",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN, constant.MANAGER),
		usecase.GetUserList(userEntity),
	)

	route.POST("",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.AddUserByRole(userEntity, systemEntity),
	)

	route.GET("/:id",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN, constant.MANAGER),
		usecase.GetUserById(userEntity),
	)

	route.DELETE("/:id",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.DeleteUserById(userEntity),
	)

	route.PUT("/:id",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.UpdateUserById(userEntity),
	)

	route.PATCH("/:id/status",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.UpdateStatusById(userEntity, sessionEntity),
	)

	route.PATCH("/:id/role",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.UpdateRoleById(userEntity, sessionEntity),
	)

	route.PATCH("/:id/set-password",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.SetPassword(userEntity, sessionEntity),
	)

	route.POST("/:id/unlock",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.UnlockUserById(userEntity, loginGuard),
	)
}
