package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/viet34tqc/demo-go-microservice/cmd/gateway/config"
	"github.com/viet34tqc/demo-go-microservice/cmd/gateway/internal/middleware"
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

	jwtMiddleware := middleware.NewJWTMiddleware(cfg.JWTSecret)
	userProxy := proxy.NewReverseProxy(cfg.UserServiceURL)
	todoProxy := proxy.NewReverseProxy(cfg.TodoServiceURL)

	api := r.Group("/api")

	// Public auth routes
	api.Any("/auth/*path", userProxy)

	// Private routes
	private := api.Group("")
	private.Use(jwtMiddleware.RequireAuth())

	private.Any("/users/*path", userProxy)
	private.Any("/todos", todoProxy)
	private.Any("/todos/*path", todoProxy)

	r.Run(":" + port)
}
