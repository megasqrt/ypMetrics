package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"
	"time"

	"ypMetrics/internal/helper"
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

const (
	defaultServerAddress   = "localhost:8080"
	defaultStoreInterval   = 300
	defaultFileStoragePath = "/tmp/metrics-db.json"
	defaultRestore         = true
	defaultDatabaseDSN     = ""
)

func parseConfig() config {
	var cfg config
	var storeInterval int

	flag.StringVar(&cfg.serverAddress, "a", defaultServerAddress, "server address")
	flag.IntVar(&storeInterval, "i", int(defaultStoreInterval), "store interval in seconds")
	flag.StringVar(&cfg.fileStoragePath, "f", defaultFileStoragePath, "file storage path")
	flag.BoolVar(&cfg.restore, "r", defaultRestore, "restore from file on start")
	flag.StringVar(&cfg.databaseDSN, "d", defaultDatabaseDSN, "database DSN")
	flag.Parse()

	viper.AutomaticEnv()
	helper.AssignFromViperIfSet(&cfg.serverAddress, "ADDRESS", viper.GetString, defaultServerAddress)
	helper.AssignFromViperIfSet(&storeInterval, "STORE_INTERVAL", viper.GetInt, defaultStoreInterval)
	helper.AssignFromViperIfSet(&cfg.fileStoragePath, "FILE_STORAGE_PATH", viper.GetString, defaultFileStoragePath)
	helper.AssignFromViperIfSet(&cfg.restore, "RESTORE", viper.GetBool, defaultRestore)
	helper.AssignFromViperIfSet(&cfg.databaseDSN, "DATABASE_DSN", viper.GetString, defaultDatabaseDSN)

	cfg.storeInterval = time.Duration(storeInterval) * time.Second

	return cfg
}

func main() {
	cfg := parseConfig()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var storage store.Storage
	var db *sql.DB
	var err error

	if cfg.databaseDSN != "" {
		log.Println("Using database storage.")
		db, err = sql.Open("pgx", cfg.databaseDSN)
		if err != nil {
			log.Fatalf("Failed to connect to the database: %v", err)
		}
		defer db.Close()

		if err = db.PingContext(ctx); err != nil && cfg.databaseDSN != "" {
			log.Fatalf("cannot connect to db: %v", err)
		}

		storage, err = store.NewDBStorage(db)
		if err != nil {
			log.Fatalf("Failed to create DB storage: %v", err)
		}
	} else {
		log.Println("Using in-memory storage.")
		memStorage, err := store.NewMemStorage(cfg.fileStoragePath, cfg.storeInterval, cfg.restore)
		if err != nil {
			log.Fatalf("Failed to create memory storage: %v", err)
		}
		storage = memStorage
		memStorage.StartPeriodicSave(ctx)
		defer memStorage.SaveOnExit()
	}

	server := services.NewMetricServer(cfg.serverAddress, storage)

	go func() {
		log.Printf("Starting server on %s", cfg.serverAddress)
		if err := server.ListenAndServe(); err != nil && err != context.Canceled {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server...")
	server.Shutdown(context.Background())

}