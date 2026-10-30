package usecase

import (
	"net/http"
	"regexp"
	"um/app/core/errs"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/app/featues/request"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var systemCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{0,49}$`)

// ListSystemCatalog returns every System the catalog describes (ADR-0012).
func ListSystemCatalog(catalog repository.ISystemCatalog) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		items, err := catalog.List()
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		ctx.JSON(http.StatusOK, items)
	}
}

// SaveSystemCatalogEntry creates or replaces the catalog entry of the code in
// the path; Clients' Systems of that code take its name.
func SaveSystemCatalogEntry(catalog repository.ISystemCatalog) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		code := ctx.Param("code")
		if !systemCodePattern.MatchString(code) {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, "system code must be upper-case letters, digits, _ or -"))
			return
		}
		req := request.SystemCatalogEntry{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, err.Error()))
			return
		}
		saved, err := catalog.Save(model.SystemCatalogEntry{SystemCode: code, SystemName: req.SystemName, WebUrl: req.WebUrl, Host: req.Host})
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		ctx.JSON(http.StatusOK, saved)
	}
}

// DeleteSystemCatalogEntry removes a code no Client has any more.
func DeleteSystemCatalogEntry(catalog repository.ISystemCatalog, systems repository.ISystem) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		code := ctx.Param("code")
		inUse, err := systems.GetSystemsByCode(code)
		if err != nil {
			logrus.Error(err)
			errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
			return
		}
		if len(inUse) > 0 {
			errs.Response(ctx, http.StatusConflict, errs.New(errs.ErrSystemInUse, "remove this system from every client first"))
			return
		}
		if err := catalog.Delete(code); err != nil {
			respondRepositoryError(ctx, err, "system")
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}
