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
)

var (
	serverAddress string
	storeInterval int
	fileStoragePath string
	restore bool
)

type config struct {
	serverAddress   string
	storeInterval   time.Duration
	fileStoragePath string
	restore         bool
}

func init() {
	flag.StringVar(&serverAddress, "a", "localhost:8080", "server address")
	flag.IntVar(&storeInterval, "i", 300, "store interval in seconds")
	flag.StringVar(&fileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&restore, "r", true, "restore from file on start")
}

func parseConfig() config {
	flag.Parse()

	viper.AutomaticEnv()
	helper.AssignFromViperIfSet(&serverAddress, "ADDRESS", viper.GetString)
	helper.AssignFromViperIfSet(&storeInterval, "STORE_INTERVAL", viper.GetInt)
	helper.AssignFromViperIfSet(&fileStoragePath, "FILE_STORAGE_PATH", viper.GetString)
	helper.AssignFromViperIfSet(&restore, "RESTORE", viper.GetBool)

	return config{
		serverAddress:   serverAddress,
		storeInterval:   time.Duration(storeInterval) * time.Second,
		fileStoragePath: fileStoragePath,
		restore:         restore,
	}
}

func main() {
	cfg := parseConfig()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	server := services.NewMetricServer(cfg.serverAddress, memStorage)

	go func() {
		log.Printf("Starting server on %s", cfg.serverAddress)
		if err := server.ListenAndServe(); err != nil && err != context.Canceled {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server...")
	server.Shutdown(context.Background())

	if fileStorage != nil {
		if err := memStorage.SaveToFile(); err != nil {
			log.Printf("Error saving metrics on shutdown: %v", err)
		} else {
			log.Println("Metrics saved successfully on shutdown.")
		}
	}
}