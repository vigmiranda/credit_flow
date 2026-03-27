package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port                  string
	DatabaseURL           string
	UploadBaseURL         string
	StorageEndpoint       string
	StoragePublicEndpoint string
	StorageAccessKey      string
	StorageSecretKey      string
	StorageBucketName     string
	StorageUseSSL         bool
}

func Load() Config {
	return Config{
		Port: getEnv("DOCUMENT_SERVICE_PORT", getEnv("PORT", "8083")),
		DatabaseURL: getEnvOrFile(
			"DOCUMENT_SERVICE_DATABASE_URL",
			"DOCUMENT_SERVICE_DATABASE_URL_FILE",
			getEnvOrFile("DATABASE_URL", "DATABASE_URL_FILE", "postgres://credit_flow:credit_flow@localhost:5432/credit_flow?sslmode=disable"),
		),
		UploadBaseURL:         getEnv("DOCUMENT_SERVICE_UPLOAD_BASE_URL", "http://localhost:4566/mock-upload"),
		StorageEndpoint:       getEnv("DOCUMENT_STORAGE_ENDPOINT", "localhost:9000"),
		StoragePublicEndpoint: getEnv("DOCUMENT_STORAGE_PUBLIC_ENDPOINT", "http://localhost:9000"),
		StorageAccessKey:      getEnv("DOCUMENT_STORAGE_ACCESS_KEY", "credit_flow"),
		StorageSecretKey:      getEnv("DOCUMENT_STORAGE_SECRET_KEY", "credit_flow"),
		StorageBucketName:     getEnv("DOCUMENT_STORAGE_BUCKET", "proposal-documents"),
		StorageUseSSL:         getEnvBool("DOCUMENT_STORAGE_USE_SSL", false),
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

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	switch strings.ToLower(value) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		return fallback
	}
}
