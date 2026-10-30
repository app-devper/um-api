package usecase

import (
	"errors"
	"net/http"
	"um/app/core/errs"
	"um/app/domain/repository"
	"um/app/featues/request"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
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

// AddSystem gives a Client a System from the catalog (ADR-0012): the name is
// the catalog's and Host defaults to the catalog's.
func AddSystem(systemEntity repository.ISystem, catalog repository.ISystemCatalog) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := request.System{}
		err := ctx.ShouldBind(&req)
		if err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}
		entry, err := catalog.Get(req.SystemCode)
		if errors.Is(err, mongo.ErrNoDocuments) {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, "system code is not in the catalog"))
			return
		}
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		if req.Host == "" {
			req.Host = entry.Host
		}
		if req.Host == "" {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, "host is required: the catalog has no default for this system"))
			return
		}

		req.CreatedBy = ctx.GetString(middlewares.UserId)
		result, err := systemEntity.CreateSystem(req, entry.SystemName)
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
