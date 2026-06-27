package config

import "os"

type Config struct {
	Port           string
	UserServiceURL string
	TodoServiceURL string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "http://localhost:8081"
	}

	todoServiceURL := os.Getenv("TODO_SERVICE_URL")
	if todoServiceURL == "" {
		todoServiceURL = "http://localhost:8082"
	}

	return Config{
		Port:           port,
		UserServiceURL: userServiceURL,
		TodoServiceURL: todoServiceURL,
	}
}
