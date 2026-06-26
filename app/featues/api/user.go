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
	secretKey string,
	userEntity repository.IUser,
	sessionEntity repository.ISession,
	systemEntity repository.ISystem,
	loginGuard repository.ILoginGuard,
) {

	route := app.Group("/user")

	route.GET("/info",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.GetUserInfo(userEntity),
	)

	route.PUT("/info",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.UpdateUserInfo(userEntity),
	)

	route.PUT("/change-password",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		usecase.ChangePassword(userEntity, sessionEntity),
	)

	route.GET("",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN, constant.MANAGER),
		usecase.GetUserList(userEntity),
	)

	route.POST("",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.AddUserByRole(userEntity, systemEntity),
	)

	route.GET("/:id",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN, constant.MANAGER),
		usecase.GetUserById(userEntity),
	)

	route.DELETE("/:id",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.DeleteUserById(userEntity),
	)

	route.PUT("/:id",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.UpdateUserById(userEntity),
	)

	route.PATCH("/:id/status",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.UpdateStatusById(userEntity, sessionEntity),
	)

	route.PATCH("/:id/role",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.UpdateRoleById(userEntity, sessionEntity),
	)

	route.PATCH("/:id/set-password",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.SetPassword(userEntity, sessionEntity),
	)

	route.POST("/:id/unlock",
		middlewares.RequireAuthenticated(secretKey),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
		usecase.UnlockUserById(userEntity, loginGuard),
	)
}
