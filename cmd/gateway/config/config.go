package config

import "os"

type Config struct {
	Port           string
	UserServiceURL string
	TodoServiceURL string
	JWTSecret      string
}

func Load() Config {
	return Config{
		Port:           getEnv("PORT", "8080"),
		UserServiceURL: getEnv("USER_SERVICE_URL", "http://localhost:8081"),
		TodoServiceURL: getEnv("TODO_SERVICE_URL", "http://localhost:8082"),
		JWTSecret:      getEnv("JWT_SECRET", "dev-secret"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
