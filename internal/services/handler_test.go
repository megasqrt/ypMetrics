package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"errors"
	"fmt"
	"encoding/json"
	"ypMetrics/internal/store"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

type MockStorage struct {
    store.Storage
	gauges    map[string]float64
    counters  map[string]int64
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
    return 0 // Дефолтное поведение
}

func (s *MockStorage) GetMetricsByTypeAndName(mName, mType string) ([]byte, error) {
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


	jsonData, err := json.Marshal(value)
		if err != nil {
			return nil,fmt.Errorf("failed to marshal metric: %w", err)
	}

	return jsonData, nil
}

func (m *MockStorage) WithGauge(name string, value float64) *MockStorage {
    m.gauges[name] = value
    return m
}

func (m *MockStorage) WithCounter(name string, value int64) *MockStorage {
    m.counters[name] = value
    return m
}

func TestMetricServer_updateHandler2(t *testing.T) {
	type want struct {
        statusCode  int
		mType string
		mName string
		mValue string
		body string
    }
	tests := []struct {
		name   string
		want   want
	}{
		{
			name: "Valid Gauge",
			want: want{
				statusCode:        http.StatusOK,
				mType: 	"gauge",
				mName: 	"Latency",
				mValue: "10",
				body: `Gauge Latency updated to`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockStorage{}
			handler := NewHandler(mock)
			request,_ := http.NewRequest(http.MethodPost, "/update", nil)
			record := httptest.NewRecorder()
			vars := map[string]string{
				"type":  tt.want.mType,
				"name":  tt.want.mName,
				"value": tt.want.mValue,
			}
			request = mux.SetURLVars(request, vars)
			t.Log(request)
            handler.updateHandler(record, request)

			assert.Equal(t, http.StatusOK, record.Code)
			assert.Contains(t, record.Body.String(), tt.want.body)
		})
	}
}

func TestGetMetricHandler(t *testing.T) {
	mock := &MockStorage{
        gauges: map[string]float64{
            "temperature": 36.6,
        },
        counters: map[string]int64{
            "requests": 42,
        },
    }
	handler := NewHandler(mock)


	type want struct {
        statusCode  int
		body string
    }
	tests := []struct {
		name         string
		mType	string
		mName  string
		want	want
	}{
		{
			name: "successful gauge request",
			mType: "gauge",
			mName: "temperature",
			want: want{
				statusCode:   http.StatusOK,
				body: `36.6`,
			},
		},
		{
			name: "successful counter request",
			mType: "counter",
			mName: "requests",
			want: want{
				statusCode:   http.StatusOK,
				body: `42`,
			},
		},
		{
			name: "missing metric",
			mType: "gauge",
			mName: "humidity",
			want: want{
				statusCode:	http.StatusNotFound,
				body: `ERROR Handler: metric 'humidity' of type 'gauge' not found`,
			}, 
		},
		{
			name: "invalid type",
			mType: "invalid",
			mName: "temperature",
			want: want{
				statusCode:	http.StatusNotFound,
				body: `ERROR Handler: invalid metric type`,
			}, 
		},
		{
			name: "missing type parameter",
			mName: "temperature",
			want: want{
				statusCode:	http.StatusNotFound,
				body: `Invalid URL format`,
			}, 
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			
			request,_ := http.NewRequest(http.MethodGet, "/value", nil)
			record := httptest.NewRecorder()
			vars := map[string]string{
				"type":  tt.mType,
				"name":  tt.mName,
			}
			request = mux.SetURLVars(request, vars)

			handler.getMetricHandler(record, request)

			assert.Equal(t, tt.want.statusCode, record.Code)
			assert.Contains(t, record.Body.String(), tt.want.body)

		})
	}
}

