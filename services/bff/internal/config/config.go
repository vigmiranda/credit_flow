package config

import "os"

type Config struct {
	Port               string
	ProposalServiceURL string
	CustomerServiceURL string
	DocumentServiceURL string
	WorkflowServiceURL string
}

func Load() Config {
	return Config{
		Port:               getEnv("BFF_PORT", getEnv("PORT", "8080")),
		ProposalServiceURL: getEnv("PROPOSAL_SERVICE_URL", "http://localhost:8081"),
		CustomerServiceURL: getEnv("CUSTOMER_SERVICE_URL", "http://localhost:8082"),
		DocumentServiceURL: getEnv("DOCUMENT_SERVICE_URL", "http://localhost:8083"),
		WorkflowServiceURL: getEnv("WORKFLOW_SERVICE_URL", "http://localhost:8084"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
