package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"creditflow/services/workflow/internal/backend"
	"creditflow/services/workflow/internal/config"
	"creditflow/services/workflow/internal/httpapi"
	"creditflow/services/workflow/internal/observability"
)

func main() {
	cfg := config.Load()
	metrics := observability.NewMetrics("workflow")
	apiHandler := metrics.Wrap(httpapi.NewServer(
		backend.NewClient(cfg.ProposalServiceURL),
		backend.NewClient(cfg.CustomerServiceURL),
		backend.NewClient(cfg.DocumentServiceURL),
		backend.NewClient(cfg.CreditAnalysisServiceURL),
		backend.NewClient(cfg.FraudAnalysisServiceURL),
		backend.NewClient(cfg.NotificationServiceURL),
		cfg.AnalysisDelay,
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
		log.Printf("workflow listening on %s", server.Addr)
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
