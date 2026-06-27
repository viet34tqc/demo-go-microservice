package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/viet34tqc/demo-go-microservice/cmd/gateway/config"
	"github.com/viet34tqc/demo-go-microservice/cmd/gateway/internal/proxy"
)

func main() {
	cfg := config.Load()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "gateway",
			"status":  "ok",
		})
	})

	// Each proxy is a Gin handler that forwards matching gateway requests to
	// the configured internal service URL.
	userProxy := proxy.NewReverseProxy(cfg.UserServiceURL)
	todoProxy := proxy.NewReverseProxy(cfg.TodoServiceURL)

	// User service routes.
	// r.Any keeps the gateway method-agnostic: GET, POST, PUT, DELETE, and
	// other HTTP methods are forwarded unchanged to user-service.
	r.Any("/api/auth/*path", userProxy)
	r.Any("/api/users/*path", userProxy)

	// Todo service routes.
	// Gin catch-all routes such as /api/todos/*path do not match /api/todos
	// without the trailing slash, so the collection route is registered too.
	r.Any("/api/todos", todoProxy)
	r.Any("/api/todos/*path", todoProxy)

	r.Run(":" + port)
}
