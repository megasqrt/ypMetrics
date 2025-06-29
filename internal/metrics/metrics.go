package metrics

import (
	"errors"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"ypMetrics/models"
)

type MemStorage struct {
	gauges         map[string]float64
	counters       map[string]int64
	fileStorage    FileStorer
	storeInterval  time.Duration
	isSyncMode     bool
}

func NewMemStorage(fs FileStorer, storeInterval time.Duration) *MemStorage {
	return &MemStorage{
		gauges:        make(map[string]float64),
		counters:      make(map[string]int64),
		fileStorage:   fs,
		storeInterval: storeInterval,
		isSyncMode:    storeInterval == 0,
	}
}

func (s *MemStorage) UpdateGauge(name string, value float64) {
	s.gauges[name] = value
}

func (s *MemStorage) UpdateCounter(name string, value int64) int64 {
	s.counters[name] += value
	return s.counters[name]
}

func (s *MemStorage) SaveToFile() error {
	if s.fileStorage != nil {
		return s.fileStorage.SaveMetrics(s)
	}
	return nil // No file storage configured
}

func (s *MemStorage) GetAllMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})
	gauges := make(map[string]float64)
	counters := make(map[string]int64)
	for k, v := range s.gauges {
		gauges[k] = v
	}
	metrics["gauges"] = gauges
	for k, v := range s.counters {
		counters[k] = v
	}
	metrics["counters"] = counters
	return metrics
}

func (s *MemStorage) getMetricValue(mName, mType string) (interface{}, bool, error) {
	switch mType {
	case "gauge":
		value, found := s.gauges[mName]
		return value, found, nil
	case "counter":
		value, found := s.counters[mName]
		return value, found, nil
	default:
		return nil, false, errors.New("invalid metric type")
	}
}

func (s *MemStorage) GetMetricsByTypeAndName(mName, mType string) ([]byte, error) {
	value, found, err := s.getMetricValue(mName, mType)
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, fmt.Errorf("metric '%s' of type '%s' not found", mName, mType)
	}

	jsonData, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metric: %w", err)
	}

	return jsonData, nil
}

func (s *MemStorage) GetJSONMetricsByTypeAndName(mName, mType string) ([]byte, error) {
	var value interface{}
	var found bool

	value, found, err := s.getMetricValue(mName, mType)
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, fmt.Errorf("metric '%s' of type '%s' not found", mName, mType)
	}

	metric := models.Metrics{ID: mName, MType: mType}
	switch mType {
	case "gauge":
		v := value.(float64)
		metric.Value = &v
	case "counter":
		v := value.(int64)
		metric.Delta = &v
	}

	jsonData, err := json.Marshal(metric)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metric object: %w", err)
	}
	return jsonData, nil
}

type FileStorer interface {
	SaveMetrics(storage *MemStorage) error
	LoadMetrics(storage *MemStorage) error
}

func (s *MemStorage) StartPeriodicSave() {
	if s.fileStorage == nil || s.isSyncMode {
		return
	}

	ticker := time.NewTicker(s.storeInterval)
	go func() {
		for range ticker.C {
			if err := s.fileStorage.SaveMetrics(s); err != nil {
				log.Printf("Error saving metrics to file: %v", err)
			}
		}
	}()
}

func (s *MemStorage) LoadFromFile() error {
	if s.fileStorage != nil {
		return s.fileStorage.LoadMetrics(s)
	}
	return nil // No file storage configured
}
