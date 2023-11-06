package api

import (
	"github.com/gin-gonic/gin"
	"um/app/core/constant"
	"um/app/domain/repository"
	"um/app/domain/usecase"
	"um/middlewares"
)

func ApplySystemAPI(
	app *gin.RouterGroup,
	systemEntity repository.ISystem,
	sessionEntity repository.ISession,
) {

	route := app.Group("system")

	route.GET("",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.RequireSession(sessionEntity),
		usecase.GetSystems(systemEntity),
	)

	route.POST("",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.RequireSession(sessionEntity),
		usecase.AddSystem(systemEntity),
	)

	route.GET("/:id",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.RequireSession(sessionEntity),
		usecase.GetSystemById(systemEntity),
	)

	route.DELETE("/:id",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.RequireSession(sessionEntity),
		usecase.DeleteSystemById(systemEntity),
	)

	route.PUT("/:id",
		middlewares.RequireAuthenticated(),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.RequireSession(sessionEntity),
		usecase.UpdateSystemById(systemEntity),
	)

}
