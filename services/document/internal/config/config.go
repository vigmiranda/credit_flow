package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port          string
	DatabaseURL   string
	UploadBaseURL string
}

func Load() Config {
	return Config{
		Port: getEnv("DOCUMENT_SERVICE_PORT", getEnv("PORT", "8083")),
		DatabaseURL: getEnvOrFile(
			"DOCUMENT_SERVICE_DATABASE_URL",
			"DOCUMENT_SERVICE_DATABASE_URL_FILE",
			getEnvOrFile("DATABASE_URL", "DATABASE_URL_FILE", "postgres://credit_flow:credit_flow@localhost:5432/credit_flow?sslmode=disable"),
		),
		UploadBaseURL: getEnv("DOCUMENT_SERVICE_UPLOAD_BASE_URL", "http://localhost:4566/mock-upload"),
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
