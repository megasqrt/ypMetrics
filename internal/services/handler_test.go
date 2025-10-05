package services

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"ypMetrics/internal/mocks"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestUpdateHandler(t *testing.T) {
	type want struct {
		statusCode int
		body       string
	}
	tests := []struct {
		name   string
		mType  string
		mName  string
		mValue string
		want   want
	}{
		{
			name:   "Valid Gauge",
			mType:  "gauge",
			mName:  "Latency",
			mValue: "10.5",
			want: want{
				statusCode: http.StatusOK,
			},
		},
		{
			name:   "Valid Counter",
			mType:  "counter",
			mName:  "Requests",
			mValue: "5",
			want: want{
				statusCode: http.StatusOK,
			},
		},
		{
			name:   "Invalid Gauge Value",
			mType:  "gauge",
			mName:  "Latency",
			mValue: "abc",
			want: want{
				statusCode: http.StatusBadRequest,
				body:       "invalid gauge value: strconv.ParseFloat: parsing \"abc\": invalid syntax\n",
			},
		},
		{
			name:   "Invalid Counter Value",
			mType:  "counter",
			mName:  "Requests",
			mValue: "abc",
			want: want{
				statusCode: http.StatusBadRequest,
				body:       "invalid counter value: strconv.ParseInt: parsing \"abc\": invalid syntax\n",
			},
		},
		{
			name:   "Invalid Metric Type",
			mType:  "unknown",
			mName:  "SomeMetric",
			mValue: "123",
			want: want{
				statusCode: http.StatusBadRequest,
				body:       "invalid metric type unknown\n",
			},
		},
		{
			name:   "Missing Metric Name",
			mType:  "gauge",
			mName:  "",
			mValue: "10",
			want: want{
				statusCode: http.StatusNotFound,
				body:       "Invalid URL format\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &mocks.MockStorage{
				Gauges:   make(map[string]float64),
				Counters: make(map[string]int64),
			}
			service := NewMetricService(mockStorage, zerolog.New(io.Discard))
			handler := NewHandler(service, zerolog.New(io.Discard))
			request, _ := http.NewRequest(http.MethodPost, "/update", nil)
			record := httptest.NewRecorder()
			vars := map[string]string{
				"type":  tt.mType,
				"name":  tt.mName,
				"value": tt.mValue,
			}
			request = mux.SetURLVars(request, vars)
			handler.updateHandler(record, request)

			assert.Equal(t, tt.want.statusCode, record.Code)
			if tt.want.body != "" {
				assert.Contains(t, record.Body.String(), tt.want.body)
			}
		})
	}
}

func TestGetMetricHandler(t *testing.T) {

	mockStorage := &mocks.MockStorage{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}
	mockStorage.WithGauge("temperature", 36.6).
		WithCounter("requests", 42)

	service := NewMetricService(mockStorage, zerolog.New(io.Discard))
	handler := NewHandler(service, zerolog.New(io.Discard))

	type want struct {
		statusCode int
		body       string
	}
	tests := []struct {
		name  string
		mType string
		mName string
		want  want
	}{
		{
			name:  "successful gauge request",
			mType: "gauge",
			mName: "temperature",
			want: want{
				statusCode: http.StatusOK,
				body:       `36.6`,
			},
		},
		{
			name:  "successful counter request",
			mType: "counter",
			mName: "requests",
			want: want{
				statusCode: http.StatusOK,
				body:       `42`,
			},
		},
		{
			name:  "missing metric",
			mType: "gauge",
			mName: "humidity",
			want: want{
				statusCode: http.StatusNotFound,
				body:       `Metric 'humidity' of type 'gauge' not found`,
			},
		},
		{
			name:  "invalid type",
			mType: "invalid",
			mName: "temperature",
			want: want{
				statusCode: http.StatusNotFound,
				body:       `Metric 'temperature' of type 'invalid' not found`,
			},
		},
		{
			name:  "missing type parameter",
			mName: "temperature",
			want: want{
				statusCode: http.StatusNotFound,
				body:       `Invalid URL format`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			request, _ := http.NewRequest(http.MethodGet, "/value", nil)
			record := httptest.NewRecorder()
			vars := map[string]string{
				"type": tt.mType,
				"name": tt.mName,
			}
			request = mux.SetURLVars(request, vars)

			handler.getMetricHandler(record, request)

			assert.Equal(t, tt.want.statusCode, record.Code)
			assert.Contains(t, record.Body.String(), tt.want.body)

		})
	}
}
