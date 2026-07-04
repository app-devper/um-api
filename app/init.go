package app

import (
	"context"
	"um/app/core/config"
	"um/app/domain/repository"
	"um/app/domain/usecase"
	"um/app/featues/api"
	"um/db"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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

	cfg, err := config.LoadAppConfig()
	if err != nil {
		logrus.Fatal(err)
	}

	resource, err := db.InitResource(cfg)
	if err != nil {
		logrus.Fatal(err)
	}
	defer resource.Close()

	r.GET("/health", usecase.Health(
		func(ctx context.Context) error { return resource.UmDb.Client().Ping(ctx, nil) },
		func(ctx context.Context) error { return resource.RdDB.Ping(ctx).Err() },
	))

	publicRoute := r.Group("/api/um/v1")

	userEntity := repository.NewUserEntity(resource)
	sessionEntity := repository.NewSessionEntity(resource)
	systemEntity := repository.NewSystemEntity(resource)
	loginGuard := repository.NewLoginGuardEntity(resource, cfg.LockoutEnabled)
	ssoEntity := repository.NewSSOTicketEntity(resource)

	api.ApplyAuthAPI(publicRoute, cfg.SecretKey, userEntity, sessionEntity, systemEntity, loginGuard, ssoEntity, resource.RdDB)
	api.ApplyUserAPI(publicRoute, cfg.SecretKey, userEntity, sessionEntity, systemEntity, loginGuard)
	api.ApplySystemAPI(publicRoute, cfg.SecretKey, systemEntity, sessionEntity, userEntity)

	r.NoRoute(middlewares.NoRoute())

	err = r.Run(cfg.ListenAddr())
	if err != nil {
		logrus.Error(err)
	}
}
