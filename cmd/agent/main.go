package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"time"
)

type MetricsAgent struct {
	serverAddress  string
	pollInterval   time.Duration
	reportInterval time.Duration
	metrics        map[string]interface{}
}

func NewMetricsAgent(serverAddress string, pollInterval, reportInterval time.Duration) *MetricsAgent {
	return &MetricsAgent{
		serverAddress:  serverAddress,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		metrics:        make(map[string]interface{}),
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
		a.metrics["RandomValue"] = rand.Float64()
		a.incrementPollCount()
	}
}

func (a *MetricsAgent) collectRuntimeMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	a.metrics["Alloc"] = float64(memStats.Alloc)
	a.metrics["BuckHashSys"] = float64(memStats.BuckHashSys)
	a.metrics["Frees"] = float64(memStats.Frees)
	a.metrics["GCCPUFraction"] = memStats.GCCPUFraction
	a.metrics["GCSys"] = float64(memStats.GCSys)
	a.metrics["HeapAlloc"] = float64(memStats.HeapAlloc)
	a.metrics["HeapIdle"] = float64(memStats.HeapIdle)
	a.metrics["HeapInuse"] = float64(memStats.HeapInuse)
	a.metrics["HeapObjects"] = float64(memStats.HeapObjects)
	a.metrics["HeapReleased"] = float64(memStats.HeapReleased)
	a.metrics["HeapSys"] = float64(memStats.HeapSys)
	a.metrics["LastGC"] = float64(memStats.LastGC)
	a.metrics["Lookups"] = float64(memStats.Lookups)
	a.metrics["MCacheInuse"] = float64(memStats.MCacheInuse)
	a.metrics["MCacheSys"] = float64(memStats.MCacheSys)
	a.metrics["MSpanInuse"] = float64(memStats.MSpanInuse)
	a.metrics["MSpanSys"] = float64(memStats.MSpanSys)
	a.metrics["Mallocs"] = float64(memStats.Mallocs)
	a.metrics["NextGC"] = float64(memStats.NextGC)
	a.metrics["NumForcedGC"] = float64(memStats.NumForcedGC)
	a.metrics["NumGC"] = float64(memStats.NumGC)
	a.metrics["OtherSys"] = float64(memStats.OtherSys)
	a.metrics["PauseTotalNs"] = float64(memStats.PauseTotalNs)
	a.metrics["StackInuse"] = float64(memStats.StackInuse)
	a.metrics["StackSys"] = float64(memStats.StackSys)
	a.metrics["Sys"] = float64(memStats.Sys)
	a.metrics["TotalAlloc"] = float64(memStats.TotalAlloc)
}

func (a *MetricsAgent) incrementPollCount() {
	if count, ok := a.metrics["PollCount"].(int64); ok {
		a.metrics["PollCount"] = count + 1
	} else {
		a.metrics["PollCount"] = int64(1)
	}
}

func (a *MetricsAgent) startReporting() {
	ticker := time.NewTicker(a.reportInterval)
	for range ticker.C {
		a.sendMetrics()
	}
}

func (a *MetricsAgent) sendMetrics() {
	for name, value := range a.metrics {
		url := a.formatMetricURL(name, value)
		if url == "" {
			continue // Пропускаем неподдерживаемые типы
		}

		resp, err := http.Post(url, "text/plain", nil)
		if err != nil {
			log.Printf("Error sending metric %s: %v", name, err)
			continue
		}
		resp.Body.Close()
	}
}

func (a *MetricsAgent) formatMetricURL(metricName string, value interface{}) string {
	switch v := value.(type) {
	case float64:
		return fmt.Sprintf("http://%s/update/gauge/%s/%f", a.serverAddress, metricName, v)
	case int64:
		return fmt.Sprintf("http://%s/update/counter/%s/%d", a.serverAddress, metricName, v)
	default:
		return ""
	}
}

func main() {
	agent := NewMetricsAgent(
		"localhost:8080",
		2*time.Second,
		10*time.Second,
	)
	agent.Run()

	select {}
}