package config

import (
	"os"
	"strconv"
	"strings"
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
	StorageWebhookMaxAge   time.Duration
	CreditWebhookMaxAge    time.Duration
	FraudWebhookMaxAge     time.Duration
	StorageProviders       []string
	CreditProviders        []string
	FraudProviders         []string
	WebhookRateLimitPrefix string
	StorageRateLimit       int
	CreditRateLimit        int
	FraudRateLimit         int
	StorageRateWindow      time.Duration
	CreditRateWindow       time.Duration
	FraudRateWindow        time.Duration
}

func Load() Config {
	defaultWebhookMaxAge := time.Duration(getEnvInt("BFF_WEBHOOK_MAX_AGE_SECONDS", 300)) * time.Second

	return Config{
		Port:                   getEnv("BFF_PORT", getEnv("PORT", "8080")),
		ProposalServiceURL:     getEnv("PROPOSAL_SERVICE_URL", "http://localhost:8081"),
		CustomerServiceURL:     getEnv("CUSTOMER_SERVICE_URL", "http://localhost:8082"),
		DocumentServiceURL:     getEnv("DOCUMENT_SERVICE_URL", "http://localhost:8083"),
		WorkflowServiceURL:     getEnv("WORKFLOW_SERVICE_URL", "http://localhost:8084"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8087"),
		WebhookSecret:          getEnv("BFF_WEBHOOK_SECRET", ""),
		WebhookMaxAge:          defaultWebhookMaxAge,
		RedisURL:               getEnv("BFF_REDIS_URL", getEnv("REDIS_URL", "")),
		WebhookReplayPrefix:    getEnv("BFF_WEBHOOK_REPLAY_PREFIX", "bff:webhook:events"),
		WebhookAuditPrefix:     getEnv("BFF_WEBHOOK_AUDIT_PREFIX", "bff:webhook:audit"),
		WebhookAuditRetention:  time.Duration(getEnvInt("BFF_WEBHOOK_AUDIT_RETENTION_SECONDS", 604800)) * time.Second,
		StorageWebhookMaxAge:   time.Duration(getEnvInt("BFF_STORAGE_WEBHOOK_MAX_AGE_SECONDS", int(defaultWebhookMaxAge.Seconds()))) * time.Second,
		CreditWebhookMaxAge:    time.Duration(getEnvInt("BFF_CREDIT_WEBHOOK_MAX_AGE_SECONDS", int(defaultWebhookMaxAge.Seconds()))) * time.Second,
		FraudWebhookMaxAge:     time.Duration(getEnvInt("BFF_FRAUD_WEBHOOK_MAX_AGE_SECONDS", int(defaultWebhookMaxAge.Seconds()))) * time.Second,
		StorageProviders:       getEnvList("BFF_ALLOWED_STORAGE_PROVIDERS"),
		CreditProviders:        getEnvList("BFF_ALLOWED_CREDIT_PROVIDERS"),
		FraudProviders:         getEnvList("BFF_ALLOWED_FRAUD_PROVIDERS"),
		WebhookRateLimitPrefix: getEnv("BFF_WEBHOOK_RATE_LIMIT_PREFIX", "bff:webhook:ratelimit"),
		StorageRateLimit:       getEnvInt("BFF_STORAGE_RATE_LIMIT", 30),
		CreditRateLimit:        getEnvInt("BFF_CREDIT_RATE_LIMIT", 30),
		FraudRateLimit:         getEnvInt("BFF_FRAUD_RATE_LIMIT", 30),
		StorageRateWindow:      time.Duration(getEnvInt("BFF_STORAGE_RATE_WINDOW_SECONDS", 60)) * time.Second,
		CreditRateWindow:       time.Duration(getEnvInt("BFF_CREDIT_RATE_WINDOW_SECONDS", 60)) * time.Second,
		FraudRateWindow:        time.Duration(getEnvInt("BFF_FRAUD_RATE_WINDOW_SECONDS", 60)) * time.Second,
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

func getEnvList(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		values = append(values, trimmed)
	}

	return values
}
