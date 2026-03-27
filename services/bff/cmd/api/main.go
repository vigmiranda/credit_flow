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
)

func main() {
	cfg := config.Load()

	server := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpapi.NewServer(
			backend.NewClient(cfg.ProposalServiceURL),
			backend.NewClient(cfg.CustomerServiceURL),
			backend.NewClient(cfg.DocumentServiceURL),
		),
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
