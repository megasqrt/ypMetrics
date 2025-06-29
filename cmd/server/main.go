package main

import (
	"os"
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"ypMetrics/internal/helper"
	"ypMetrics/internal/metrics"
	"ypMetrics/internal/services"
	"ypMetrics/internal/store"
)

func main() {
	viper.AutomaticEnv()

	var storeInterval int
	var fileStoragePath string
	var restore bool

	flag.IntVar(&storeInterval, "i", 300, "store interval in seconds")
	flag.StringVar(&fileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&restore, "r", true, "restore from file on start")
	var serverAddress string
	flag.StringVar(&serverAddress, "a", "localhost:8080", "server address")
	flag.Parse()

	helper.AssignIfNotEmpty(&storeInterval, viper.GetInt("STORE_INTERVAL"))
	helper.AssignIfNotEmpty(&fileStoragePath, viper.GetString("FILE_STORAGE_PATH"))
	if os.Getenv("RESTORE") != "" {
		restore = viper.GetBool("RESTORE")
	}

	var fileStorage metrics.FileStorer
	if fileStoragePath != "" {
		fileStorage = store.NewFileStorage(fileStoragePath)
	}
	memStorage := metrics.NewMemStorage(fileStorage, time.Duration(storeInterval)*time.Second)

	if fileStorage != nil && restore {
		if err := memStorage.LoadFromFile(); err != nil {
			log.Printf("Warning: could not load metrics from file: %v", err)
		}
	}

	memStorage.StartPeriodicSave()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := services.NewMetricServer(serverAddress, memStorage)

	go func() {
		log.Printf("Starting server on %s", serverAddress)
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