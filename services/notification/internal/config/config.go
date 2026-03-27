package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port          string
	DatabaseURL   string
	EncryptionKey string
}

func Load() Config {
	return Config{
		Port: getEnv("NOTIFICATION_SERVICE_PORT", getEnv("PORT", "8087")),
		DatabaseURL: getEnvOrFile(
			"NOTIFICATION_SERVICE_DATABASE_URL",
			"NOTIFICATION_SERVICE_DATABASE_URL_FILE",
			getEnvOrFile("DATABASE_URL", "DATABASE_URL_FILE", "postgres://credit_flow:credit_flow@localhost:5432/credit_flow?sslmode=disable"),
		),
		EncryptionKey: getEnvOrFile(
			"NOTIFICATION_SERVICE_ENCRYPTION_KEY",
			"NOTIFICATION_SERVICE_ENCRYPTION_KEY_FILE",
			"credit-flow-local-notification-key",
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
