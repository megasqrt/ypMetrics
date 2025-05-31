package main

import (
	"ypMetrics/internal/metrics"
	"ypMetrics/internal/services"
)

func main() {
	storage := metrics.NewMemStorage()
	services.NewMetricServer(storage)
}
