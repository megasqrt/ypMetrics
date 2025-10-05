package agent

import (
	"math/rand"
	"runtime"
	"sync"

	"fmt"
	"log"
	"ypMetrics/internal/helper"
	"ypMetrics/models"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type MetricCollector struct {
	mu        sync.Mutex
	pollCount int64
	metrics   map[string]models.Metrics
}

func NewMetricCollector() *MetricCollector {
	return &MetricCollector{
		metrics: make(map[string]models.Metrics),
	}
}

func (c *MetricCollector) Poll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pollCount++
	c.updateCounter("PollCount", c.pollCount)
	c.updateGauge("RandomValue", rand.Float64())
	c.pollRuntimeMetrics()
}

func (c *MetricCollector) GetMetrics() []models.Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]models.Metrics, 0, len(c.metrics))
	for _, m := range c.metrics {
		result = append(result, m)
	}
	return result
}

func (c *MetricCollector) updateCounter(id string, value int64) {
	c.metrics[id] = models.Metrics{
		ID:    id,
		MType: models.Counter,
		Delta: helper.Ptr(value),
	}
}

func (c *MetricCollector) updateGauge(id string, value float64) {
	c.metrics[id] = models.Metrics{
		ID:    id,
		MType: models.Gauge,
		Value: helper.Ptr(value),
	}
}

func (c *MetricCollector) pollRuntimeMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.updateGauge("Alloc", float64(memStats.Alloc))
	c.updateGauge("BuckHashSys", float64(memStats.BuckHashSys))
	c.updateGauge("Frees", float64(memStats.Frees))
	c.updateGauge("GCCPUFraction", memStats.GCCPUFraction)
	c.updateGauge("GCSys", float64(memStats.GCSys))
	c.updateGauge("HeapAlloc", float64(memStats.HeapAlloc))
	c.updateGauge("HeapIdle", float64(memStats.HeapIdle))
	c.updateGauge("HeapInuse", float64(memStats.HeapInuse))
	c.updateGauge("HeapObjects", float64(memStats.HeapObjects))
	c.updateGauge("HeapReleased", float64(memStats.HeapReleased))
	c.updateGauge("HeapSys", float64(memStats.HeapSys))
	c.updateGauge("LastGC", float64(memStats.LastGC))
	c.updateGauge("Lookups", float64(memStats.Lookups))
	c.updateGauge("MCacheInuse", float64(memStats.MCacheInuse))
	c.updateGauge("MCacheSys", float64(memStats.MCacheSys))
	c.updateGauge("MSpanInuse", float64(memStats.MSpanInuse))
	c.updateGauge("MSpanSys", float64(memStats.MSpanSys))
	c.updateGauge("Mallocs", float64(memStats.Mallocs))
	c.updateGauge("NextGC", float64(memStats.NextGC))
	c.updateGauge("NumForcedGC", float64(memStats.NumForcedGC))
	c.updateGauge("NumGC", float64(memStats.NumGC))
	c.updateGauge("OtherSys", float64(memStats.OtherSys))
	c.updateGauge("PauseTotalNs", float64(memStats.PauseTotalNs))
	c.updateGauge("StackInuse", float64(memStats.StackInuse))
	c.updateGauge("StackSys", float64(memStats.StackSys))
	c.updateGauge("Sys", float64(memStats.Sys))
	c.updateGauge("TotalAlloc", float64(memStats.TotalAlloc))
}

func (c *MetricCollector) PollGopsutil() {
	c.mu.Lock()
	defer c.mu.Unlock()

	vm, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting memory stats: %v", err)
	} else {
		c.updateGauge("TotalMemory", float64(vm.Total))
		c.updateGauge("FreeMemory", float64(vm.Free))
	}

	cpuPercentages, err := cpu.Percent(0, true)
	if err != nil {
		log.Printf("Error getting cpu stats: %v", err)
	} else {
		for i, p := range cpuPercentages {
			metricName := fmt.Sprintf("CPUutilization%d", i+1)
			c.updateGauge(metricName, p)
		}
	}
}
