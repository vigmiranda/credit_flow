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
	"creditflow/services/notification/internal/repository/postgres"
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

	repo := postgres.NewNotificationRepository(db)
	if err := repo.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewServer(repo),
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
