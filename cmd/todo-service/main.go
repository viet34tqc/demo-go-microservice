package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/handler"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/internal/config"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/internal/db"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/middleware"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/repository"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "todo-service",
			"status":  "ok",
		})
	})

	cfg := config.Load()

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	todoRepo := repository.NewTodoRepository(database)
	todoHandler := handler.NewTodoHandler(todoRepo)

	todos := r.Group("/todos")
	todos.Use(middleware.RequireUserID())
	{
		todos.POST("", todoHandler.Create)
		todos.GET("", todoHandler.List)
		todos.GET("/:id", todoHandler.GetByID)
		todos.PUT("/:id", todoHandler.Update)
		todos.DELETE("/:id", todoHandler.Delete)
	}

	r.Run(":" + port)
}
