package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/handler"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/internal/config"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/internal/db"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/middleware"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/repository"
	todoservice "github.com/viet34tqc/demo-go-microservice/cmd/todo-service/service"
	"google.golang.org/grpc"

	grpcserver "github.com/viet34tqc/demo-go-microservice/cmd/todo-service/internal/grpc"
	todopb "github.com/viet34tqc/demo-go-microservice/gen/go/todo/v1"
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

	todoService := todoservice.NewTodoService(database)
	go startTodoGRPCServer(todoService)

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

func startTodoGRPCServer(todoService *todoservice.TodoService) {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen todo grpc server: %v", err)
	}

	grpcServer := grpc.NewServer()

	todopb.RegisterTodoServiceServer(
		grpcServer,
		grpcserver.NewTodoGRPCServer(todoService),
	)

	log.Println("todo-service gRPC server is running on :50051")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve todo grpc server: %v", err)
	}
}
