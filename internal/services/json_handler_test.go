package services

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ypMetrics/internal/helper"
	"ypMetrics/internal/mocks"
	"ypMetrics/internal/services/middlewares"
	"ypMetrics/models"

	"github.com/rs/zerolog"
)

func TestUpdateMetricJSON(t *testing.T) {
	tests := []struct {
		name           string
		initialStorage *mocks.MockStorage
		requestMetric  models.Metrics
		expectedMetric models.Metrics
		expectedStatus int
	}{
		{
			name: "update gauge",
			initialStorage: &mocks.MockStorage{
				Gauges:   make(map[string]float64),
				Counters: make(map[string]int64),
			},
			requestMetric: models.Metrics{
				ID:    "TestGauge",
				MType: "gauge",
				Value: helper.Float64Ptr(123.45),
			},
			expectedMetric: models.Metrics{
				ID:    "TestGauge",
				MType: "gauge",
				Value: helper.Float64Ptr(123.45),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "update new counter",
			initialStorage: &mocks.MockStorage{
				Gauges:   make(map[string]float64),
				Counters: make(map[string]int64),
			},
			requestMetric: models.Metrics{
				ID:    "TestCounter",
				MType: "counter",
				Delta: helper.Int64Ptr(10),
			},
			expectedMetric: models.Metrics{
				ID:    "TestCounter",
				MType: "counter",
				Delta: helper.Int64Ptr(10),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "update existing counter",
			initialStorage: &mocks.MockStorage{
				Gauges:   make(map[string]float64),
				Counters: map[string]int64{"TestCounter": 20},
			},
			requestMetric: models.Metrics{
				ID:    "TestCounter",
				MType: "counter",
				Delta: helper.Int64Ptr(5),
			},
			expectedMetric: models.Metrics{
				ID:    "TestCounter",
				MType: "counter",
				Delta: helper.Int64Ptr(25), // 20 + 5
			},
			expectedStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем логгер, который ничего не выводит, для чистоты тестов
			service := NewMetricService(tt.initialStorage, zerolog.New(io.Discard))
			ipCheckMiddleware := middlewares.NewIPCheckMiddleware("")
			handler := NewHandler(service, zerolog.New(io.Discard), ipCheckMiddleware)

			body, err := json.Marshal(tt.requestMetric)
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(body))
			require.NoError(t, err)
			req = req.WithContext(context.Background())

			respRecord := httptest.NewRecorder()
			handler.UpdateMetricJSON(respRecord, req)

			assert.Equal(t, tt.expectedStatus, respRecord.Code)

			if tt.expectedStatus == http.StatusOK {
				expectedBody, err := json.Marshal(tt.expectedMetric)
				require.NoError(t, err)
				assert.JSONEq(t, string(expectedBody), respRecord.Body.String())
			}
		})
	}
}

func TestUpdateMetricJSON_InvalidData(t *testing.T) {
	tests := []struct {
		name     string
		request  interface{}
		expected int
	}{
		{
			name:     "Empty ID",
			request:  map[string]interface{}{"MType": "gauge", "Value": 123.45},
			expected: http.StatusBadRequest,
		},
		{
			name:     "Invalid MType",
			request:  map[string]interface{}{"ID": "test", "MType": "invalid", "Value": 123.45},
			expected: http.StatusBadRequest,
		},
		{
			name:     "Missing Value for Gauge",
			request:  map[string]interface{}{"ID": "test", "MType": "gauge"},
			expected: http.StatusBadRequest,
		},
		{
			name:     "Missing Delta for Counter",
			request:  map[string]interface{}{"ID": "test", "MType": "counter"},
			expected: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req, err := http.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(body))
			require.NoError(t, err)
			req = req.WithContext(context.Background())
			resp := httptest.NewRecorder()

			service := NewMetricService(&mocks.MockStorage{}, zerolog.New(io.Discard))
			ipCheckMiddleware := middlewares.NewIPCheckMiddleware("")
			handler := NewHandler(service, zerolog.New(io.Discard), ipCheckMiddleware)
			handler.UpdateMetricJSON(resp, req)

			assert.Equal(t, tt.expected, resp.Code)
		})
	}
}

func TestGetMetricJSON(t *testing.T) {
	storage := &mocks.MockStorage{
		Gauges:   map[string]float64{"TestGauge": 99.9},
		Counters: map[string]int64{"TestCounter": 42},
	}

	tests := []struct {
		name           string
		requestMetric  models.Metrics
		expectedMetric models.Metrics
		expectedStatus int
	}{
		{
			name: "get existing gauge",
			requestMetric: models.Metrics{
				ID:    "TestGauge",
				MType: "gauge",
			},
			expectedMetric: models.Metrics{
				ID:    "TestGauge",
				MType: "gauge",
				Value: helper.Float64Ptr(99.9),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "get existing counter",
			requestMetric: models.Metrics{
				ID:    "TestCounter",
				MType: "counter",
			},
			expectedMetric: models.Metrics{
				ID:    "TestCounter",
				MType: "counter",
				Delta: helper.Int64Ptr(42),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "get nonexistent metric",
			requestMetric: models.Metrics{
				ID:    "NotFound",
				MType: "gauge",
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMetricService(storage, zerolog.New(io.Discard))
			ipCheckMiddleware := middlewares.NewIPCheckMiddleware("")
			handler := NewHandler(service, zerolog.New(io.Discard), ipCheckMiddleware)
			body, err := json.Marshal(tt.requestMetric)
			require.NoError(t, err)
			req, err := http.NewRequest(http.MethodPost, "/value/", bytes.NewBuffer(body))
			require.NoError(t, err)
			respRecord := httptest.NewRecorder()
			handler.GetMetricJSON(respRecord, req)
			assert.Equal(t, tt.expectedStatus, respRecord.Code)
			if tt.expectedStatus == http.StatusOK {
				expectedBody, err := json.Marshal(tt.expectedMetric)
				require.NoError(t, err)
				assert.JSONEq(t, string(expectedBody), respRecord.Body.String())
			}
		})
	}
}
