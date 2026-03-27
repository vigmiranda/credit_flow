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

	"creditflow/services/document/internal/config"
	"creditflow/services/document/internal/httpapi"
	"creditflow/services/document/internal/repository/postgres"
	miniostorage "creditflow/services/document/internal/storage/minio"
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

	repo := postgres.NewDocumentRepository(db)
	if err := repo.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	storage, err := miniostorage.New(miniostorage.Config{
		Endpoint:       cfg.StorageEndpoint,
		PublicEndpoint: cfg.StoragePublicEndpoint,
		AccessKey:      cfg.StorageAccessKey,
		SecretKey:      cfg.StorageSecretKey,
		BucketName:     cfg.StorageBucketName,
		UseSSL:         cfg.StorageUseSSL,
	})
	if err != nil {
		log.Fatalf("new storage: %v", err)
	}
	if err := storage.EnsureBucket(ctx); err != nil {
		log.Fatalf("ensure bucket: %v", err)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewServer(repo, storage.PublicURL(""), storage),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("document service listening on %s", server.Addr)
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
