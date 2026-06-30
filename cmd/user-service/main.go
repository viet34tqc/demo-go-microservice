package main

import (
	"log"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/viet34tqc/demo-go-microservice/cmd/user-service/internal/config"
	"github.com/viet34tqc/demo-go-microservice/cmd/user-service/internal/db"
	grpcserver "github.com/viet34tqc/demo-go-microservice/cmd/user-service/internal/grpc"
	"github.com/viet34tqc/demo-go-microservice/cmd/user-service/internal/handler"
	"github.com/viet34tqc/demo-go-microservice/cmd/user-service/internal/middleware"
	"github.com/viet34tqc/demo-go-microservice/cmd/user-service/internal/service"
	userpb "github.com/viet34tqc/demo-go-microservice/gen/go/user/v1"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	userService := service.NewUserService(database, cfg.JWTSecret)
	go startUserGRPCServer(userService)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "user-service",
			"status":  "ok",
		})
	})

	authHandler := handler.NewAuthHandler(database, cfg)

	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/login", authHandler.Login)

	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		protected.GET("/users/me", authHandler.Me)
	}

	addr := ":" + cfg.Port

	log.Printf("user-service running on %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func startUserGRPCServer(userService *service.UserService) {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen user grpc server: %v", err)
	}

	grpcServer := grpc.NewServer()

	userpb.RegisterUserServiceServer(
		grpcServer,
		grpcserver.NewUserGRPCServer(userService),
	)

	log.Println("user-service gRPC server is running on :50051")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve user grpc server: %v", err)
	}
}
