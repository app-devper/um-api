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
		limit(ctx, rdb, fmt.Sprintf("rate_limit:%s:%s", ctx.FullPath(), ctx.ClientIP()), maxAttempts, window)
	}
}

// RateLimiterPerUser limits requests per signed-in User, for routes behind
// RequireSession. The service sees every request from the gateway's address,
// so a limit per IP there would be one limit shared by all Users.
func RateLimiterPerUser(rdb *redis.Client, maxAttempts int, window time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limit(ctx, rdb, fmt.Sprintf("rate_limit:%s:user:%s", ctx.FullPath(), ctx.GetString(UserId)), maxAttempts, window)
	}
}

// limit counts the request under key and refuses it with 429 once the key has
// had more than maxAttempts requests in the window.
func limit(ctx *gin.Context, rdb *redis.Client, key string, maxAttempts int, window time.Duration) {
	count, err := rdb.Incr(context.Background(), key).Result()
	if err != nil {
		errs.Response(ctx, http.StatusServiceUnavailable, errs.New(errs.ErrInternal, "service temporarily unavailable"))
		return
	}

	if count == 1 {
		rdb.Expire(context.Background(), key, window)
	}

	if count > int64(maxAttempts) {
		errs.Response(ctx, http.StatusTooManyRequests, errs.New(errs.ErrRateLimited, "too many requests, please try again later"))
		return
	}

	ctx.Next()
}
