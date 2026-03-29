package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"creditflow/services/bff/internal/backend"
	"creditflow/services/bff/internal/config"
	"creditflow/services/bff/internal/httpapi"
	"creditflow/services/bff/internal/observability"
)

func main() {
	cfg := config.Load()
	metrics := observability.NewMetrics("bff")
	replayStore, err := httpapi.NewRedisWebhookReplayStore(cfg.RedisURL, cfg.WebhookReplayPrefix)
	if err != nil || cfg.RedisURL == "" {
		replayStore = httpapi.NewMemoryWebhookReplayStore()
	}
	auditStore, err := httpapi.NewRedisWebhookAuditStore(cfg.RedisURL, cfg.WebhookAuditPrefix, cfg.WebhookAuditRetention)
	if err != nil || cfg.RedisURL == "" {
		auditStore = httpapi.NewMemoryWebhookAuditStore()
	}
	apiHandler := metrics.Wrap(httpapi.NewServerWithDependencies(
		backend.NewClient(cfg.ProposalServiceURL),
		backend.NewClient(cfg.CustomerServiceURL),
		backend.NewClient(cfg.DocumentServiceURL),
		backend.NewClient(cfg.WorkflowServiceURL),
		backend.NewClient(cfg.NotificationServiceURL),
		cfg.WebhookSecret,
		cfg.WebhookMaxAge,
		replayStore,
		auditStore,
		metrics,
	))

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.Handle("/", apiHandler)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("bff listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
}
