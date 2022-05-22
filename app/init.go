package app

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"os"
	"um/app/domain/repository"
	"um/app/featues/api"
	"um/db"
	"um/middlewares"
)

type Routes struct {
}

func (app Routes) StartGin() {
	r := gin.New()

	err := r.SetTrustedProxies(nil)
	if err != nil {
		logrus.Error(err)
	}

	r.Use(gin.Logger())
	r.Use(middlewares.NewRecovery())
	r.Use(middlewares.NewCors([]string{"*"}))

	resource, err := db.InitResource()
	if err != nil {
		logrus.Error(err)
	}
	defer resource.Close()

	publicRoute := r.Group("/api/um/v1")

	userEntity := repository.NewUserEntity(resource)
	sessionEntity := repository.NewSessionEntity(resource)
	systemEntity := repository.NewSystemEntity(resource)

	api.ApplyAuthAPI(publicRoute, userEntity, sessionEntity)
	api.ApplyUserAPI(publicRoute, userEntity, sessionEntity, systemEntity)
	api.ApplyAdminUserAPI(publicRoute, userEntity, sessionEntity)
	api.ApplySuperUserAPI(publicRoute, userEntity, sessionEntity)
	api.ApplySystemAPI(publicRoute, systemEntity, sessionEntity)

	r.NoRoute(middlewares.NoRoute())

	err = r.Run(":" + os.Getenv("PORT"))
	if err != nil {
		logrus.Error(err)
	}
}
