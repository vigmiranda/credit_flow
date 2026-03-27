package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
}

func Load() Config {
	return Config{
		Port: getEnv("NOTIFICATION_SERVICE_PORT", getEnv("PORT", "8087")),
		DatabaseURL: getEnv(
			"NOTIFICATION_SERVICE_DATABASE_URL",
			getEnv("DATABASE_URL", "postgres://credit_flow:credit_flow@localhost:5432/credit_flow?sslmode=disable"),
		),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
