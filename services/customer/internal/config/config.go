package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port        string
	DatabaseURL string
}

func Load() Config {
	return Config{
		Port: getEnv("CUSTOMER_SERVICE_PORT", getEnv("PORT", "8082")),
		DatabaseURL: getEnvOrFile(
			"CUSTOMER_SERVICE_DATABASE_URL",
			"CUSTOMER_SERVICE_DATABASE_URL_FILE",
			getEnvOrFile("DATABASE_URL", "DATABASE_URL_FILE", "postgres://credit_flow:credit_flow@localhost:5432/credit_flow?sslmode=disable"),
		),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvOrFile(key, fileKey, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	if filePath := strings.TrimSpace(os.Getenv(fileKey)); filePath != "" {
		raw, err := os.ReadFile(filePath)
		if err != nil {
			panic(fmt.Sprintf("read secret file %s: %v", filePath, err))
		}
		return strings.TrimSpace(string(raw))
	}

	return fallback
}
