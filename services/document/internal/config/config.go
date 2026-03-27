package config

import "os"

type Config struct {
	Port          string
	DatabaseURL   string
	UploadBaseURL string
}

func Load() Config {
	return Config{
		Port: getEnv("DOCUMENT_SERVICE_PORT", getEnv("PORT", "8083")),
		DatabaseURL: getEnv(
			"DOCUMENT_SERVICE_DATABASE_URL",
			getEnv("DATABASE_URL", "postgres://credit_flow:credit_flow@localhost:5432/credit_flow?sslmode=disable"),
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
