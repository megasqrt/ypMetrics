package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"
	"time"

	"ypMetrics/internal/helper"
	"ypMetrics/internal/metrics"
	"ypMetrics/internal/services"
	"ypMetrics/internal/store"

	"github.com/spf13/viper"

	"database/sql"
    _ "github.com/jackc/pgx/v5/stdlib"
)

type config struct {
	serverAddress   string
	storeInterval   time.Duration
	fileStoragePath string
	restore         bool
	databaseDSN     string	
}

func parseConfig() config {
	var cfg config
	var storeInterval int

	flag.StringVar(&cfg.serverAddress, "a", "localhost:8080", "server address")
	flag.IntVar(&storeInterval, "i", 300, "store interval in seconds")
	flag.StringVar(&cfg.fileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&cfg.restore, "r", true, "restore from file on start")
	flag.StringVar(&cfg.databaseDSN, "d", "host=127.0.0.1 user=metric password=metric dbname=metric sslmode=disable", "database DSN")

	flag.Parse()

	viper.AutomaticEnv()
	helper.AssignFromViperIfSet(&cfg.serverAddress, "ADDRESS", viper.GetString)
	helper.AssignFromViperIfSet(&storeInterval, "STORE_INTERVAL", viper.GetInt)
	helper.AssignFromViperIfSet(&cfg.fileStoragePath, "FILE_STORAGE_PATH", viper.GetString)
	helper.AssignFromViperIfSet(&cfg.restore, "RESTORE", viper.GetBool)
	helper.AssignFromViperIfSet(&cfg.databaseDSN, "DATABASE_DSN", viper.GetString)
	
	return cfg
}

func main() {
	cfg := parseConfig()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := sql.Open("pgx", cfg.databaseDSN)
    if err != nil {
        log.Fatalf("Failed to connect to the database: %v", err)
    }
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("cannot connect to db: %v", err)
	}
    defer db.Close()

	var fileStorage metrics.FileStorer
	if cfg.fileStoragePath != "" {
		fileStorage = store.NewFileStorage(cfg.fileStoragePath)
	}
	memStorage := metrics.NewMemStorage(fileStorage, cfg.storeInterval)

	if fileStorage != nil && cfg.restore {
		if err := memStorage.LoadFromFile(); err != nil {
			log.Printf("Warning: could not load metrics from file: %v", err)
		}
	}

	memStorage.StartPeriodicSave(ctx)

	server := services.NewMetricServer(cfg.serverAddress, memStorage, db)

	go func() {
		log.Printf("Starting server on %s", cfg.serverAddress)
		if err := server.ListenAndServe(); err != nil && err != context.Canceled {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server...")
	server.Shutdown(context.Background())

	log.Println("Closing database connection...")
	if err := db.Close(); err != nil {
		log.Printf("Error closing database connection: %v", err)
	}
	log.Println("Database connection closed.")

	if fileStorage != nil {
		if err := memStorage.SaveToFile(); err != nil {
			log.Printf("Error saving metrics on shutdown: %v", err)
		} else {
			log.Println("Metrics saved successfully on shutdown.")
		}
	}
}