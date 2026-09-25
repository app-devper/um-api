package usecase

import (
	"errors"
	"net/http"
	"um/app/core/errs"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func respondRepositoryError(ctx *gin.Context, err error, resource string) {
	switch {
	case errors.Is(err, primitive.ErrInvalidHex):
		errs.Response(ctx, http.StatusBadRequest, errs.New(errs.ErrBadRequest, "invalid id"))
	case errors.Is(err, mongo.ErrNoDocuments):
		errs.Response(ctx, http.StatusNotFound, errs.New(errs.ErrNotFound, resource+" not found"))
	default:
		errs.Response(ctx, http.StatusInternalServerError, errs.New(errs.ErrInternal, "internal server error"))
	}
}
