package middlewares

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// NewCors return new gin handler fuc to handle CORS request
func NewCors(allowedOrigins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders: []string{
			"Origin", "Host",
			"Content-Type", "Content-Length",
			"Accept-Encoding", "Accept-Language", "Accept",
			"X-CSRF-Token", "Authorization", "X-Requested-With", "X-Access-Token",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	})
}
