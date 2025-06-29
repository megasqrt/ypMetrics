package mocks

import (
	"errors"
	"fmt"
	"encoding/json"
	"ypMetrics/internal/store"
)

type MockStorage struct {
    store.Storage
	Gauges    map[string]float64
    Counters  map[string]int64
	UpdateGaugeFunc func(name string, value float64) 
	UpdateCounterFunc func(name string, value int64) int64 
	GetMetricsByTypeAndNameFunc func(mName, mType string) ([]byte, error) 
}

func (m *MockStorage) UpdateGauge(name string,value float64){
    if m.UpdateGaugeFunc != nil {
        m.UpdateGaugeFunc(name,value)
    }
   
}

func (m *MockStorage) UpdateCounter(name string, value int64) int64 {
    if m.UpdateCounterFunc != nil {
        return m.UpdateCounterFunc(name, value)
    }
    return 0 // Default
}

func (m *MockStorage) GetMetricsByTypeAndName(mName, mType string) ([]byte, error) {
	var value interface{}
	var found bool

	switch mType {
	case "gauge":
		value, found = m.Gauges[mName]
	case "counter":
		value, found = m.Counters[mName]
	default:
		return nil, errors.New("invalid metric type")
	}

	if !found {
		return nil, fmt.Errorf("metric '%s' of type '%s' not found", mName, mType)
	}


	jsonData, err := json.Marshal(value)
		if err != nil {
			return nil,fmt.Errorf("failed to marshal metric: %w", err)
	}

	return jsonData, nil
}

func (m *MockStorage) WithGauge(name string, value float64) *MockStorage {
    m.Gauges[name] = value
    return m
}

func (m *MockStorage) WithCounter(name string, value int64) *MockStorage {
    m.Counters[name] = value
    return m
}