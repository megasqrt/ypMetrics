package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	"ypMetrics/models"
)

// MemStorage is an in-memory metric storage.
type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64

	filePath      string
	storeInterval time.Duration
	syncSave      bool
}

// NewMemStorage creates a new MemStorage.
func NewMemStorage(filePath string, storeInterval time.Duration, restore bool) (*MemStorage, error) {
	s := &MemStorage{
		gauges:        make(map[string]float64),
		counters:      make(map[string]int64),
		filePath:      filePath,
		storeInterval: storeInterval,
		syncSave:      storeInterval == 0,
	}

	if restore && filePath != "" {
		if err := s.loadFromFile(); err != nil {
			// Log the error but don't fail, as the file might not exist on first run.
			log.Printf("Warning: could not load metrics from file: %v", err)
		}
	}

	return s, nil
}

func (s *MemStorage) UpdateGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
	if s.syncSave {
		s.saveToFile()
	}
}

func (s *MemStorage) UpdateCounter(name string, value int64) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
	newValue := s.counters[name]
	if s.syncSave {
		s.saveToFile()
	}
	return newValue
}

func (s *MemStorage) GetAllMetrics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	gaugesCopy := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		gaugesCopy[k] = v
	}

	countersCopy := make(map[string]int64, len(s.counters))
	for k, v := range s.counters {
		countersCopy[k] = v
	}

	return map[string]interface{}{
		"gauges":   gaugesCopy,
		"counters": countersCopy,
	}
}

func (s *MemStorage) GetMetricsByTypeAndName(mName, mType string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch mType {
	case models.Gauge:
		if val, ok := s.gauges[mName]; ok {
			return []byte(fmt.Sprintf("%g", val)), nil
		}
	case models.Counter:
		if val, ok := s.counters[mName]; ok {
			return []byte(fmt.Sprintf("%d", val)), nil
		}
	default:
		return nil, fmt.Errorf("invalid metric type")
	}
	return nil, fmt.Errorf("metric '%s' of type '%s' not found", mName, mType)
}

func (s *MemStorage) GetJSONMetricsByTypeAndName(mName, mType string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var metric models.Metrics
	metric.ID = mName
	metric.MType = mType

	switch mType {
	case models.Gauge:
		if val, ok := s.gauges[mName]; ok {
			metric.Value = &val
			return json.Marshal(metric)
		}
	case models.Counter:
		if val, ok := s.counters[mName]; ok {
			metric.Delta = &val
			return json.Marshal(metric)
		}
	}
	return nil, fmt.Errorf("metric not found")
}

func (s *MemStorage) UpdateMetricsBatch(metrics []models.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value != nil {
				s.gauges[m.ID] = *m.Value
			}
		case models.Counter:
			if m.Delta != nil {
				s.counters[m.ID] += *m.Delta
			}
		}
	}

	if s.syncSave {
		return s.saveToFile()
	}

	return nil
}

func (s *MemStorage) Ping(ctx context.Context) error {
	// In-memory storage is always available
	return nil
}

// StartPeriodicSave starts a goroutine to periodically save metrics to a file.
func (s *MemStorage) StartPeriodicSave(ctx context.Context) {
	if s.filePath == "" || s.storeInterval <= 0 {
		return
	}

	ticker := time.NewTicker(s.storeInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := s.saveToFile(); err != nil {
					log.Printf("Error saving metrics to file: %v", err)
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// SaveOnExit saves metrics to a file before the application exits.
func (s *MemStorage) SaveOnExit() {
	if s.filePath != "" {
		if err := s.saveToFile(); err != nil {
			log.Printf("Error saving metrics on shutdown: %v", err)
		} else {
			log.Println("Metrics saved successfully on shutdown.")
		}
	}
}

func (s *MemStorage) saveToFile() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.filePath == "" {
		return nil
	}

	data, err := json.Marshal(s.GetAllMetrics())
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0666)
}

func (s *MemStorage) loadFromFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var allMetrics struct {
		Gauges   map[string]float64 `json:"gauges"`
		Counters map[string]int64   `json:"counters"`
	}

	if err := json.Unmarshal(data, &allMetrics); err != nil {
		return err
	}

	s.gauges = allMetrics.Gauges
	s.counters = allMetrics.Counters

	return nil
}
