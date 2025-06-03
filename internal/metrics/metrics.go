package metrics

import (
	"errors"
	"fmt"
	"encoding/json"
)

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

type StorageInterface interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64) int64
	GetAllMetrics() map[string]interface{}
}

func (s *MemStorage) UpdateGauge(name string, value float64) {
	s.gauges[name] = value
}

func (s *MemStorage) UpdateCounter(name string, value int64) int64 {
	s.counters[name] += value
	return s.counters[name]
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

func (s *MemStorage) GetMetricsByTypeAndName(mName, mType string) ([]byte, error) {
	var value interface{}
	var found bool

	switch mType {
	case "gauge":
		value, found = s.gauges[mName]
	case "counter":
		value, found = s.counters[mName]
	default:
		return nil, errors.New("invalid metric type")
	}

	if !found {
		return nil, fmt.Errorf("metric '%s' of type '%s' not found", mName, mType)
	}

	// вдруг пригодиться 
	// response := struct {
    //     Value interface{} `json:"value"`
    // }{
    //     Value: value,
    // }

	jsonData, err := json.Marshal(value)
		if err != nil {
			return nil,fmt.Errorf("failed to marshal metric: %w", err)
	}

	return jsonData, nil
}
