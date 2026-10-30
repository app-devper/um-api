package app

import (
	"context"
	"um/app/core/config"
	"um/app/domain/passkey"
	"um/app/domain/repository"
	"um/app/domain/session"
	"um/app/domain/systemcatalog"
	"um/app/domain/usecase"
	"um/app/domain/useradmin"
	"um/app/featues/api"
	"um/app/featues/request"
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

	r.Use(middlewares.NewGatewayHost(cfg.GatewayHosts))

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
	catalogEntity := repository.NewSystemCatalogEntity(resource)
	seedSystemCatalog(systemEntity, catalogEntity)
	loginGuard := repository.NewLoginGuardEntity(resource, cfg.LockoutEnabled)
	ssoEntity := repository.NewSSOTicketEntity(resource)
	sessions := session.NewManager(cfg.SecretKey, sessionEntity, userEntity, session.WithLogin(systemEntity, loginGuard), session.WithCatalog(catalogEntity))
	passkeyEntity := repository.NewPasskeyEntity(resource)
	passkeys, err := passkey.New(passkey.Config{RPID: cfg.WebAuthnRPID, RPName: cfg.WebAuthnRPName, Origins: cfg.WebAuthnOrigin},
		passkeyEntity, repository.NewPasskeyCeremonyEntity(resource), userEntity, sessions)
	if err != nil {
		logrus.Fatal(err)
	}
	admin := useradmin.New(userEntity, sessions, systemEntity, loginGuard)
	useradmin.WithPasskeys(passkeyEntity)(admin)

	protectedRoute := publicRoute.Group("", usecase.RequireSession(sessions))

	api.ApplyAuthAPI(publicRoute, protectedRoute, sessions, userEntity, sessionEntity, systemEntity, loginGuard, ssoEntity, resource.RdDB)
	api.ApplyPasskeyAPI(publicRoute, protectedRoute, passkeys, resource.RdDB)
	api.ApplyUserAPI(protectedRoute, userEntity, admin)
	api.ApplySystemAPI(protectedRoute, systemEntity, catalogEntity)

	r.NoRoute(middlewares.NoRoute())

	err = r.Run(cfg.ListenAddr())
	if err != nil {
		logrus.Error(err)
	}
}

// seedSystemCatalog adds a catalog entry for every System code Clients have
// but the catalog lacks (ADR-0012); existing entries are left as they are.
func seedSystemCatalog(systems repository.ISystem, catalog repository.ISystemCatalog) {
	all, err := systems.GetSystems(request.GetSystems{})
	if err != nil {
		logrus.Errorf("system catalog seed: list systems: %v", err)
		return
	}
	added, err := catalog.InsertMissing(systemcatalog.Seed(all))
	if err != nil {
		logrus.Errorf("system catalog seed: %v", err)
		return
	}
	if added > 0 {
		logrus.Infof("system catalog seed: added %d entries", added)
	}
}
