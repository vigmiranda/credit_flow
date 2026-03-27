package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                     string
	ProposalServiceURL       string
	CustomerServiceURL       string
	DocumentServiceURL       string
	CreditAnalysisServiceURL string
	FraudAnalysisServiceURL  string
	NotificationServiceURL   string
	RedisURL                 string
	QueueName                string
	WorkerCount              int
	MaxRetries               int
	QueuePollTimeout         time.Duration
	AnalysisDelay            time.Duration
}

func Load() Config {
	return Config{
		Port:                     getEnv("WORKFLOW_SERVICE_PORT", getEnv("PORT", "8084")),
		ProposalServiceURL:       getEnv("WORKFLOW_SERVICE_PROPOSAL_URL", getEnv("PROPOSAL_SERVICE_URL", "http://localhost:8081")),
		CustomerServiceURL:       getEnv("WORKFLOW_SERVICE_CUSTOMER_URL", getEnv("CUSTOMER_SERVICE_URL", "http://localhost:8082")),
		DocumentServiceURL:       getEnv("WORKFLOW_SERVICE_DOCUMENT_URL", getEnv("DOCUMENT_SERVICE_URL", "http://localhost:8083")),
		CreditAnalysisServiceURL: getEnv("WORKFLOW_SERVICE_CREDIT_URL", getEnv("CREDIT_ANALYSIS_SERVICE_URL", "http://localhost:8085")),
		FraudAnalysisServiceURL:  getEnv("WORKFLOW_SERVICE_FRAUD_URL", getEnv("FRAUD_ANALYSIS_SERVICE_URL", "http://localhost:8086")),
		NotificationServiceURL:   getEnv("WORKFLOW_SERVICE_NOTIFICATION_URL", getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8087")),
		RedisURL:                 getEnv("WORKFLOW_REDIS_URL", getEnv("REDIS_URL", "")),
		QueueName:                getEnv("WORKFLOW_QUEUE_NAME", "workflow:proposal-analysis"),
		WorkerCount:              getEnvInt("WORKFLOW_WORKER_COUNT", 1),
		MaxRetries:               getEnvInt("WORKFLOW_MAX_RETRIES", 2),
		QueuePollTimeout:         time.Duration(getEnvInt("WORKFLOW_QUEUE_POLL_TIMEOUT_MS", 1000)) * time.Millisecond,
		AnalysisDelay:            time.Duration(getEnvInt("WORKFLOW_ANALYSIS_DELAY_MS", 250)) * time.Millisecond,
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
