package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                   string
	ProposalServiceURL     string
	CustomerServiceURL     string
	DocumentServiceURL     string
	WorkflowServiceURL     string
	NotificationServiceURL string
	WebhookSecret          string
	WebhookMaxAge          time.Duration
	RedisURL               string
	WebhookReplayPrefix    string
	WebhookAuditPrefix     string
	WebhookAuditRetention  time.Duration
}

func Load() Config {
	return Config{
		Port:                   getEnv("BFF_PORT", getEnv("PORT", "8080")),
		ProposalServiceURL:     getEnv("PROPOSAL_SERVICE_URL", "http://localhost:8081"),
		CustomerServiceURL:     getEnv("CUSTOMER_SERVICE_URL", "http://localhost:8082"),
		DocumentServiceURL:     getEnv("DOCUMENT_SERVICE_URL", "http://localhost:8083"),
		WorkflowServiceURL:     getEnv("WORKFLOW_SERVICE_URL", "http://localhost:8084"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8087"),
		WebhookSecret:          getEnv("BFF_WEBHOOK_SECRET", ""),
		WebhookMaxAge:          time.Duration(getEnvInt("BFF_WEBHOOK_MAX_AGE_SECONDS", 300)) * time.Second,
		RedisURL:               getEnv("BFF_REDIS_URL", getEnv("REDIS_URL", "")),
		WebhookReplayPrefix:    getEnv("BFF_WEBHOOK_REPLAY_PREFIX", "bff:webhook:events"),
		WebhookAuditPrefix:     getEnv("BFF_WEBHOOK_AUDIT_PREFIX", "bff:webhook:audit"),
		WebhookAuditRetention:  time.Duration(getEnvInt("BFF_WEBHOOK_AUDIT_RETENTION_SECONDS", 604800)) * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
