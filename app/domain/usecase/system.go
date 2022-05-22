package usecase

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"um/app/domain/repository"
	"um/app/featues/request"
)

func GetSystem(systemEntity repository.ISystem) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		system := ctx.GetString("System")
		clientId := ctx.GetString("ClientId")
		result, err := systemEntity.GetSystem(clientId, system)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func GetSystems(systemEntity repository.ISystem) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		result, err := systemEntity.GetSystemAll()
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func AddSystem(systemEntity repository.ISystem) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.System{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userId := ctx.GetString("UserId")

		req.CreatedBy = userId
		result, err := systemEntity.CreateSystem(req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func GetSystemById(systemEntity repository.ISystem) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		result, err := systemEntity.GetSystemById(id)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func DeleteSystemById(systemEntity repository.ISystem) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		result, err := systemEntity.RemoveSystemById(id)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func UpdateSystemById(systemEntity repository.ISystem) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.UpdateSystem{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		userId := ctx.GetString("UserId")
		id := ctx.Param("id")
		req.UpdatedBy = userId
		result, err := systemEntity.UpdateSystemById(id, req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}
