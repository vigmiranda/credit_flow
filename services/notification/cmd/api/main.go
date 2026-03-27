package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"creditflow/services/notification/internal/config"
	"creditflow/services/notification/internal/httpapi"
	"creditflow/services/notification/internal/observability"
	"creditflow/services/notification/internal/repository/postgres"
	"creditflow/services/notification/internal/security"
	smtpsender "creditflow/services/notification/internal/sender/smtp"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	crypto, err := security.NewCipher(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("load encryption key: %v", err)
	}

	repo := postgres.NewNotificationRepository(db, crypto)
	if err := repo.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}
	sender := smtpsender.New(smtpsender.Config{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
	})

	metrics := observability.NewMetrics("notification")
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.Handle("/", metrics.Wrap(httpapi.NewServer(repo, sender)))

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("notification service listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-shutdownCtx.Done()

	finalCtx, finalCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer finalCancel()

	if err := server.Shutdown(finalCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
}
