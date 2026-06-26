package db

import (
	"fmt"

	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/internal/config"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Bangkok",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPass,
		cfg.DBName,
		cfg.DBPort,
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := database.AutoMigrate(&model.Todo{}); err != nil {
		return nil, err
	}

	return database, nil
}
