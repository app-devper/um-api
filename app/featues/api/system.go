package api

import (
	"um/app/core/constant"
	"um/app/domain/repository"
	"um/app/domain/usecase"
	"um/middlewares"

	"github.com/gin-gonic/gin"
)

func ApplySystemAPI(
	app *gin.RouterGroup,
	systemEntity repository.ISystem,
	sessionEntity repository.ISession,
	userEntity repository.IUser,
) {

	route := app.Group("system")

	route.GET("",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.GetSystems(systemEntity),
	)

	route.POST("",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.AddSystem(systemEntity),
	)

	route.GET("/:id",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.GetSystemById(systemEntity),
	)

	route.DELETE("/:id",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.DeleteSystemById(systemEntity),
	)

	route.PUT("/:id",
		middlewares.RequireAuthenticated(),
		usecase.RequireSession(sessionEntity, userEntity),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.UpdateSystemById(systemEntity),
	)

}
