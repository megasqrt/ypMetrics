package main

import (
	"bytes"
	"compress/gzip"
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

var runtimeMetricNames = map[string]struct{}{
	"Alloc": {}, "BuckHashSys": {}, "Frees": {}, "GCCPUFraction": {}, "GCSys": {},
	"HeapAlloc": {}, "HeapIdle": {}, "HeapInuse": {}, "HeapObjects": {}, "HeapReleased": {},
	"HeapSys": {}, "LastGC": {}, "Lookups": {}, "MCacheInuse": {}, "MCacheSys": {},
	"MSpanInuse": {}, "MSpanSys": {}, "Mallocs": {}, "NextGC": {}, "NumForcedGC": {},
	"NumGC": {}, "OtherSys": {}, "PauseTotalNs": {}, "StackInuse": {}, "StackSys": {},
	"Sys": {}, "TotalAlloc": {},
}

func isRuntimeMetric(name string) bool {
	_, ok := runtimeMetricNames[name]
	return ok
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
		if a.pingServer() {
			a.sendMetricsBatch()
		} else {
			a.sendMetrics()
		}
	}
}

func sendGzippedJSON(url string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга данных: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(jsonData); err != nil {
		gz.Close()
		return fmt.Errorf("ошибка сжатия данных: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("ошибка закрытия gzip writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("сервер вернул статус не-OK: %s", resp.Status)
	}
	return nil
}

func (a *MetricsAgent) sendMetrics() {
	for _, m := range a.metrics {
		url := fmt.Sprintf("http://%s/update/", a.serverAddress)
		if err := sendGzippedJSON(url, m); err != nil {
			log.Printf("Ошибка отправки метрики %s: %v", m.ID, err)
		}
	}
}

func (a *MetricsAgent) sendMetricsBatch() {
	url := fmt.Sprintf("http://%s/updates/", a.serverAddress)
	if err := sendGzippedJSON(url, a.metrics); err != nil {
		log.Printf("Ошибка отправки пакета метрик: %v", err)
	}
}

func (a *MetricsAgent) pingServer() bool {
	resp, err := http.Get(fmt.Sprintf("http://%s/ping", a.serverAddress))
	if err != nil {
		log.Printf("Server ping failed: %v. Falling back to single metric updates.", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

var (
	serverAddress  string
	reportInterval int
	pollInterval   int
)

type config struct {
	serverAddress  string
	pollInterval   time.Duration
	reportInterval time.Duration
}
	
func init() {
	flag.StringVar(&serverAddress, "a", "localhost:8080", "server adress")
	flag.IntVar(&reportInterval, "r", 10, "report interval")
	flag.IntVar(&pollInterval, "p", 2, "poll interval")
}

func parseConfig() config {
	flag.Parse()

	viper.AutomaticEnv()

	helper.AssignFromViperIfSet(&serverAddress, "ADDRESS", viper.GetString)
	helper.AssignFromViperIfSet(&reportInterval, "REPORT_INTERVAL", viper.GetInt)
	helper.AssignFromViperIfSet(&pollInterval, "POLL_INTERVAL", viper.GetInt)
	
	if !govalidator.IsURL(serverAddress) {
		log.Fatalf("некорректный URL %s", serverAddress)
	}

	return config{
		serverAddress:  serverAddress,
		reportInterval: time.Duration(reportInterval) * time.Second,
		pollInterval:   time.Duration(pollInterval) * time.Second,
	}
}

func main() {
	cfg := parseConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Printf("start push metric to %s \n", cfg.serverAddress)

		agent := NewMetricsAgent(
			cfg.serverAddress,
			cfg.pollInterval,
			cfg.reportInterval,
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