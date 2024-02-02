package usecase

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"um/app/domain/repository"
	"um/app/featues/request"
	"um/middlewares"
)

func NotifyPosProductLotsExpire(systemEntity repository.ISystem) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		systemCode := "POS"
		result, err := systemEntity.GetSystemsByCode(systemCode)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		path := "/api/pos/v1/products/lots/expire-notify"
		for _, item := range result {
			_, _ = middlewares.NotifyMassage(item.Host + path)
		}
		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}

func GetSystem(systemEntity repository.ISystem) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		systemCode := ctx.GetString(middlewares.System)
		clientId := ctx.GetString(middlewares.ClientId)
		result, err := systemEntity.GetSystem(clientId, systemCode)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}

func GetSystems(systemEntity repository.ISystem) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.GetSystems{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := systemEntity.GetSystems(req)
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

		userId := ctx.GetString(middlewares.UserId)

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
		userId := ctx.GetString(middlewares.UserId)
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
