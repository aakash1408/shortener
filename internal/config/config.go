package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port    int
	Env     string
	BaseURL string

	DatabaseURL string

	JWTSecret   string
	LogFilePath string
	LogLevel    string

	RateLimitRPS int
}

func Load() (Config, error) {
	portStr := os.Getenv("PORT")
	port := 8080
	if portStr != "" {
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return Config{}, fmt.Errorf("invalid PORT value %q: %w", portStr, err)
		}
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	logFilePath := os.Getenv("LOG_FILE")

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	rateLimitRPS := 10
	rateLimitStr := os.Getenv("RATE_LIMIT_RPS")
	if rateLimitStr != "" {
		var err error
		rateLimitRPS, err = strconv.Atoi(rateLimitStr)
		if err != nil {
			return Config{}, fmt.Errorf("invalid RATE_LIMIT_RPS value %q: %w", rateLimitStr, err)
		}
	}

	return Config{
		Port:         port,
		Env:          env,
		BaseURL:      baseURL,
		DatabaseURL:  databaseURL,
		JWTSecret:    jwtSecret,
		LogFilePath:  logFilePath,
		LogLevel:     logLevel,
		RateLimitRPS: rateLimitRPS,
	}, nil
}
