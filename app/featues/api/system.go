package api

import (
	"um/app/core/constant"
	"um/app/domain/repository"
	"um/app/domain/usecase"
	"um/middlewares"

	"github.com/gin-gonic/gin"
)

func ApplySystemAPI(
	protected *gin.RouterGroup,
	systemEntity repository.ISystem,
) {

	route := protected.Group("system", middlewares.RequireAuthorization(constant.SUPER))

	route.GET("", usecase.GetSystems(systemEntity))
	route.POST("", usecase.AddSystem(systemEntity))
	route.GET("/:id", usecase.GetSystemById(systemEntity))
	route.DELETE("/:id", usecase.DeleteSystemById(systemEntity))
	route.PUT("/:id", usecase.UpdateSystemById(systemEntity))
}
