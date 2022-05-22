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
		middlewares.RequireAuthenticated(sessionEntity),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.GetSystems(systemEntity),
	)

	route.POST("",
		middlewares.RequireAuthenticated(sessionEntity),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.AddSystem(systemEntity),
	)

	route.GET("/:id",
		middlewares.RequireAuthenticated(sessionEntity),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.GetSystemById(systemEntity),
	)

	route.DELETE("/:id",
		middlewares.RequireAuthenticated(sessionEntity),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.DeleteSystemById(systemEntity),
	)

	route.PUT("/:id",
		middlewares.RequireAuthenticated(sessionEntity),
		middlewares.RequireAuthorization(constant.SUPER),
		usecase.UpdateSystemById(systemEntity),
	)

}
