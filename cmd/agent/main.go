package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/spf13/viper"

	"ypMetrics/internal/helper"
	"ypMetrics/models"
)

type MetricsAgent struct {
	serverAddress  string
	pollInterval   time.Duration
	reportInterval time.Duration
	metrics        []models.Metrics
}

func NewMetricsAgent(serverAddress string, pollInterval, reportInterval time.Duration) *MetricsAgent {
	return &MetricsAgent{
		serverAddress:  serverAddress,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		// metrics:        make(map[string]interface{}),
	}
}

func (a *MetricsAgent) Run() {
	go a.startPolling()
	go a.startReporting()
}

func (a *MetricsAgent) startPolling() {
	ticker := time.NewTicker(a.pollInterval)
	for range ticker.C {
		a.collectRuntimeMetrics()
		a.metrics= append(a.metrics, models.Metrics{
            ID:    "RandomValue",
            MType: "gauge",
            Value: ptrFloat64(rand.Float64()),
        })
		a.incrementPollCount()
	}
}

func (a *MetricsAgent) collectRuntimeMetrics() {
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)

    metrics := []models.Metrics{
		{ID: "Alloc", MType: "gauge", Value: ptrFloat64(float64(memStats.Alloc))},
        {ID: "BuckHashSys", MType: "gauge", Value: ptrFloat64(float64(memStats.BuckHashSys))},
        {ID: "Frees", MType: "gauge", Value: ptrFloat64(float64(memStats.Frees))},
        {ID: "GCCPUFraction", MType: "gauge", Value: ptrFloat64(memStats.GCCPUFraction)},
        {ID: "GCSys", MType: "gauge", Value: ptrFloat64(float64(memStats.GCSys))},
        {ID: "HeapAlloc", MType: "gauge", Value: ptrFloat64(float64(memStats.HeapAlloc))},
        {ID: "HeapIdle", MType: "gauge", Value: ptrFloat64(float64(memStats.HeapIdle))},
        {ID: "HeapInuse", MType: "gauge", Value: ptrFloat64(float64(memStats.HeapInuse))},
        {ID: "HeapObjects", MType: "gauge", Value: ptrFloat64(float64(memStats.HeapObjects))},
        {ID: "HeapReleased", MType: "gauge", Value: ptrFloat64(float64(memStats.HeapReleased))},
        {ID: "HeapSys", MType: "gauge", Value: ptrFloat64(float64(memStats.HeapSys))},
        {ID: "LastGC", MType: "gauge", Value: ptrFloat64(float64(memStats.LastGC))},
        {ID: "Lookups", MType: "gauge", Value: ptrFloat64(float64(memStats.Lookups))},
        {ID: "MCacheInuse", MType: "gauge", Value: ptrFloat64(float64(memStats.MCacheInuse))},
        {ID: "MCacheSys", MType: "gauge", Value: ptrFloat64(float64(memStats.MCacheSys))},
        {ID: "MSpanInuse", MType: "gauge", Value: ptrFloat64(float64(memStats.MSpanInuse))},
        {ID: "MSpanSys", MType: "gauge", Value: ptrFloat64(float64(memStats.MSpanSys))},
        {ID: "Mallocs", MType: "gauge", Value: ptrFloat64(float64(memStats.Mallocs))},
        {ID: "NextGC", MType: "gauge", Value: ptrFloat64(float64(memStats.NextGC))},
        {ID: "NumForcedGC", MType: "gauge", Value: ptrFloat64(float64(memStats.NumForcedGC))},
        {ID: "NumGC", MType: "gauge", Value: ptrFloat64(float64(memStats.NumGC))},
        {ID: "OtherSys", MType: "gauge", Value: ptrFloat64(float64(memStats.OtherSys))},
        {ID: "PauseTotalNs", MType: "gauge", Value: ptrFloat64(float64(memStats.PauseTotalNs))},
        {ID: "StackInuse", MType: "gauge", Value: ptrFloat64(float64(memStats.StackInuse))},
        {ID: "StackSys", MType: "gauge", Value: ptrFloat64(float64(memStats.StackSys))},
        {ID: "Sys", MType: "gauge", Value: ptrFloat64(float64(memStats.Sys))},
        {ID: "TotalAlloc", MType: "gauge", Value: ptrFloat64(float64(memStats.TotalAlloc))},
    }

    // Заменяем метрики runtime (очищаем старые и добавляем новые)
    a.metrics = filterOutRuntimeMetrics(a.metrics)
    a.metrics = append(a.metrics, metrics...)
}

func filterOutRuntimeMetrics(metrics []models.Metrics) []models.Metrics {
    var result []models.Metrics
    for _, m := range metrics {
        if !isRuntimeMetric(m.ID) {
            result = append(result, m)
        }
    }
    return result
}

func isRuntimeMetric(name string) bool {
    runtimeMetrics := []string{
        "Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
        "HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
        "HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
        "MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
        "NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
        "Sys", "TotalAlloc",
    }
    for _, m := range runtimeMetrics {
        if m == name {
            return true
        }
    }
    return false
}

func (a *MetricsAgent) incrementPollCount() {
	for i := range a.metrics {
		if a.metrics[i].ID == "PollCount" && a.metrics[i].MType == "counter" {
			if a.metrics[i].Delta != nil {
				*a.metrics[i].Delta++
			} else {
				a.metrics[i].Delta = ptrInt64(1)
			}
			return
		}
	}
	// если не нашли, добавляем новую
	a.metrics = append(a.metrics, models.Metrics{ID: "PollCount", MType: "counter", Delta: ptrInt64(1)})
}


func (a *MetricsAgent) startReporting() {
	ticker := time.NewTicker(a.reportInterval)
	for range ticker.C {
		a.sendMetrics()
	}
}

func (a *MetricsAgent) sendMetrics() {
	for _, m := range a.metrics {
		url := fmt.Sprintf("http://%s/update/", a.serverAddress)
		
		data, err := json.Marshal(m)
        if err != nil {
            continue
        }

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("Error sending metric %s: %v", m.ID, err)
			continue
		}
		resp.Body.Close()
	}
}

var (
	serverAddress  string
	reportInterval int
	pollInterval   int
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	viper.AutomaticEnv() 
    
    envAddress := viper.GetString("ADDRESS") 
	envReportInterval := viper.GetInt("REPORT_INTERVAL") 
	envPollInterval := viper.GetInt("POLL_INTERVAL") 

	flag.StringVar(&serverAddress, "a", "localhost:8080", "server adress")
	flag.IntVar(&reportInterval, "r", 10, "report interval")
	flag.IntVar(&pollInterval, "p", 2, "poll interval")

	flag.Parse()

	helper.AssignIfNotEmpty(&serverAddress, envAddress)
	helper.AssignIfNotEmpty(&reportInterval, envReportInterval)
	helper.AssignIfNotEmpty(&pollInterval, envPollInterval)

	if !govalidator.IsURL(serverAddress) {
    	log.Fatalf("некорректный URL %s",serverAddress)
	}

	go func() {
		fmt.Printf("start push metric to %s", serverAddress)

		agent := NewMetricsAgent(
			serverAddress,
			time.Duration(pollInterval)*time.Second,
			time.Duration(reportInterval)*time.Second,
		)
		agent.Run()
		<-ctx.Done()
	}()

	sig := <-sigChan
	log.Printf("Получен сигнал: %v\n", sig)
	cancel()
}

func ptrFloat64(v float64) *float64 {
    return &v
}
func ptrInt64(v int64) *int64 {
    return &v
}