package mocks

import (
	"encoding/json"
	"errors"
	"fmt"

	"ypMetrics/internal/store"
	"ypMetrics/models"
)

type MockStorage struct {
	Gauges                      map[string]float64
	Counters                    map[string]int64
	UpdateGaugeFunc             func(name string, value float64)
	UpdateCounterFunc           func(name string, value int64) int64
	GetMetricsByTypeAndNameFunc func(mName, mType string) ([]byte, error)
	GetJSONMetricsByTypeAndNameFunc func(mName, mType string) ([]byte, error)
}

var _ store.Storage = (*MockStorage)(nil)

func (m *MockStorage) UpdateGauge(name string, value float64) {
	if m.UpdateGaugeFunc != nil {
		m.UpdateGaugeFunc(name, value)
		return
	}
	if m.Gauges == nil {
		m.Gauges = make(map[string]float64)
	}
	m.Gauges[name] = value
}

func (m *MockStorage) UpdateCounter(name string, value int64) int64 {
	if m.UpdateCounterFunc != nil {
		return m.UpdateCounterFunc(name, value)
	}
	if m.Counters == nil {
		m.Counters = make(map[string]int64)
	}
	m.Counters[name] += value
	return m.Counters[name]
}

func (m *MockStorage) getMetricValue(mName, mType string) (interface{}, bool, error) {
	switch mType {
	case "gauge":
		if m.Gauges == nil {
			return nil, false, nil
		}
		value, found := m.Gauges[mName]
		return value, found, nil
	case "counter":
		if m.Counters == nil {
			return nil, false, nil
		}
		value, found := m.Counters[mName]
		return value, found, nil
	default:
		return nil, false, errors.New("invalid metric type")
	}
}

func (m *MockStorage) GetMetricsByTypeAndName(mName, mType string) ([]byte, error) {
	value, found, err := m.getMetricValue(mName, mType)
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

func (m *MockStorage) GetJSONMetricsByTypeAndName(mName, mType string) ([]byte, error) {
	value, found, err := m.getMetricValue(mName, mType)
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

func (m *MockStorage) GetAllMetrics() map[string]interface{} {
	return map[string]interface{}{
		"gauges":   m.Gauges,
		"counters": m.Counters,
	}
}

func (m *MockStorage) WithGauge(name string, value float64) *MockStorage {
	if m.Gauges == nil {
		m.Gauges = make(map[string]float64)
	}
	m.Gauges[name] = value
	return m
}

func (m *MockStorage) WithCounter(name string, value int64) *MockStorage {
	if m.Counters == nil {
		m.Counters = make(map[string]int64)
	}
	m.Counters[name] = value
	return m
}