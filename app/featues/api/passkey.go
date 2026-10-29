package api

import (
	"time"
	"um/app/domain/passkey"
	"um/app/domain/usecase"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// ApplyPasskeyAPI registers Passkey login on the public group and Passkey
// management on the protected group (ADR-0010).
func ApplyPasskeyAPI(public *gin.RouterGroup, protected *gin.RouterGroup, passkeys *passkey.Service, rdb *redis.Client) {
	open := public.Group("auth/passkey")
	open.POST("/login/options", middlewares.RateLimiter(rdb, 10, 1*time.Minute), usecase.PasskeyLoginOptions(passkeys))
	open.POST("/login", middlewares.RateLimiter(rdb, 10, 1*time.Minute), usecase.PasskeyLogin(passkeys))

	route := protected.Group("auth/passkeys")
	route.POST("/options", middlewares.RateLimiter(rdb, 5, 1*time.Minute), usecase.PasskeyRegistrationOptions(passkeys))
	route.POST("", usecase.RegisterPasskey(passkeys))
	route.GET("", usecase.ListPasskeys(passkeys))
	route.DELETE("/:id", usecase.DeletePasskey(passkeys))
}
