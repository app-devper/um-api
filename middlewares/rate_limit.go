package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"um/app/core/errs"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// RateLimiter returns a middleware that limits requests per IP using Redis.
// maxAttempts: max requests allowed in the window.
// window: time window for the rate limit.
func RateLimiter(rdb *redis.Client, maxAttempts int, window time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()
		key := fmt.Sprintf("rate_limit:%s:%s", ctx.FullPath(), ip)

		count, err := rdb.Incr(context.Background(), key).Result()
		if err != nil {
			// fail-open: allow request if Redis is unavailable
			ctx.Next()
			return
		}

		rdb.Expire(context.Background(), key, window)

		if count > int64(maxAttempts) {
			errs.Response(ctx, http.StatusTooManyRequests, errs.New(errs.ErrRateLimited, "too many requests, please try again later"))
			return
		}

		ctx.Next()
	}
}
