package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"
	"time"

	"net/http"
	_ "net/http/pprof"
	"ypMetrics/internal/helper"
	"ypMetrics/internal/misc"
	"ypMetrics/internal/services"
	"ypMetrics/internal/store"

	"github.com/spf13/viper"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	defaultServerAddress   = "localhost:8080"
	defaultStoreInterval   = 300
	defaultFileStoragePath = "/tmp/metrics-db.json"
	defaultRestore         = true
	defaultDatabaseDSN     = ""
	defaultHashKey         = ""
)

func parseConfig() misc.Config {
	var cfg misc.Config
	var storeInterval int

	flag.StringVar(&cfg.ServerAddress, "a", defaultServerAddress, "server address")
	flag.IntVar(&storeInterval, "i", int(defaultStoreInterval), "store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", defaultFileStoragePath, "file storage path")
	flag.BoolVar(&cfg.Restore, "r", defaultRestore, "restore from file on start")
	flag.StringVar(&cfg.DatabaseDSN, "d", defaultDatabaseDSN, "database DSN")
	flag.StringVar(&cfg.HashKey, "k", defaultHashKey, "key for hashing")
	flag.Parse()

	viper.AutomaticEnv()
	helper.AssignFromViperIfSet(&cfg.ServerAddress, "ADDRESS", viper.GetString, defaultServerAddress)
	helper.AssignFromViperIfSet(&storeInterval, "STORE_INTERVAL", viper.GetInt, defaultStoreInterval)
	helper.AssignFromViperIfSet(&cfg.FileStoragePath, "FILE_STORAGE_PATH", viper.GetString, defaultFileStoragePath)
	helper.AssignFromViperIfSet(&cfg.Restore, "RESTORE", viper.GetBool, defaultRestore)
	helper.AssignFromViperIfSet(&cfg.DatabaseDSN, "DATABASE_DSN", viper.GetString, defaultDatabaseDSN)
	helper.AssignFromViperIfSet(&cfg.HashKey, "KEY", viper.GetString, defaultHashKey)

	cfg.StoreInterval = time.Duration(storeInterval) * time.Second

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

	if cfg.DatabaseDSN != "" {
		log.Println("Using database storage.")
		db, err = sql.Open("pgx", cfg.DatabaseDSN)
		if err != nil {
			log.Fatalf("Failed to connect to the database: %v", err)
		}
		defer db.Close()

		if err = db.PingContext(ctx); err != nil && cfg.DatabaseDSN != "" {
			log.Fatalf("cannot connect to db: %v", err)
		}

		storage, err = store.NewDBStorage(db)
		if err != nil {
			log.Fatalf("Failed to create DB storage: %v", err)
		}
		log.Println("Successfully create database storage.")
	} else {
		log.Println("Using in-memory storage.")
		memStorage, err := store.NewMemStorage(cfg.FileStoragePath, cfg.StoreInterval, cfg.Restore)
		if err != nil {
			log.Fatalf("Failed to create memory storage: %v", err)
		}
		storage = memStorage
		memStorage.StartPeriodicSave(ctx)
		defer memStorage.SaveOnExit()
	}

	server := services.NewMetricServer(cfg, storage)

	go func() {
		log.Println("Starting pprof server on :6060")
		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Fatalf("pprof server error: %v", err)
		}
	}()

	go func() {
		log.Printf("Starting server on %s", cfg.ServerAddress)
		if err := server.ListenAndServe(); err != nil && err != context.Canceled {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server...")
	server.Shutdown(context.Background())

}