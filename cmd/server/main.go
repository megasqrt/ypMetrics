package main

import (
	"ypMetrics/internal/metrics"
	"ypMetrics/internal/services"
	"ypMetrics/internal/store"
	"os"
	"fmt"
)

func main() {
	storage :=  initStorage()
	err:=services.NewMetricServer(storage)
	if err != nil{
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
	}
}

func initStorage() store.Storage {
	//refactored after added some storage
    switch os.Getenv("STORAGE_TYPE") {
    case "memory":
        return metrics.NewMemStorage()
    default:
        return metrics.NewMemStorage()
    }
}