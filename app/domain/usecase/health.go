package usecase

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Pinger func(ctx context.Context) error

func Health(mongoPing, redisPing Pinger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		pingCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
		defer cancel()

		mongoStatus := "ok"
		if err := mongoPing(pingCtx); err != nil {
			mongoStatus = err.Error()
		}

		redisStatus := "ok"
		if err := redisPing(pingCtx); err != nil {
			redisStatus = err.Error()
		}

		status := http.StatusOK
		overall := "ok"
		if mongoStatus != "ok" || redisStatus != "ok" {
			status = http.StatusServiceUnavailable
			overall = "degraded"
		}

		ctx.JSON(status, gin.H{
			"status": overall,
			"checks": gin.H{
				"mongo": mongoStatus,
				"redis": redisStatus,
			},
		})
	}
}
