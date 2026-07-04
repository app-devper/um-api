package usecase

import (
	"net/http"
	"um/app/core/errs"
	"um/app/domain/repository"
	"um/app/featues/request"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func GetSystem(systemEntity repository.ISystem) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		systemCode := ctx.GetString(middlewares.System)
		clientId := ctx.GetString(middlewares.ClientId)
		result, err := systemEntity.GetSystem(clientId, systemCode)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "system")
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
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		result, err := systemEntity.GetSystems(req)
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
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
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}

		userId := ctx.GetString(middlewares.UserId)

		req.CreatedBy = userId
		result, err := systemEntity.CreateSystem(req)
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
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
			logrus.Error(err)
			respondRepositoryError(ctx, err, "system")
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
			logrus.Error(err)
			respondRepositoryError(ctx, err, "system")
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
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}
		userId := ctx.GetString(middlewares.UserId)
		id := ctx.Param("id")
		req.UpdatedBy = userId
		result, err := systemEntity.UpdateSystemById(id, req)
		if err != nil {
			logrus.Error(err)
			respondRepositoryError(ctx, err, "system")
			return
		}
		ctx.JSON(http.StatusOK, result)
	}
}
