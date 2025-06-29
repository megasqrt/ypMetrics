package services

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ypMetrics/models"
	"ypMetrics/internal/mocks"
)

func TestUpdateMetricJSON_Success(t *testing.T) {
	type want struct {
        statusCode  int
		mType string
		mName string
		mValue string
		metric models.Metrics
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
				metric: models.Metrics{
					ID:    "test-gauge",
					MType: "gauge",
					Value: float64Ptr(123.45),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &mocks.MockStorage{}
			handler := &Handler{mockStorage}
			body, _ := json.Marshal(tt.want.metric)
			req, err := http.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(body))
			require.NoError(t, err)
			
			respRecord := httptest.NewRecorder()
			handler.UpdateMetricJSON(respRecord, req)
			
			assert.Equal(t, http.StatusOK, respRecord.Code)
			assert.JSONEq(t, respRecord.Body.String(), string(body))
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
			req, _ := http.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(body))
			resp := httptest.NewRecorder()
			
			handler := &Handler{storage: &mocks.MockStorage{}}
			handler.UpdateMetricJSON(resp, req)
			
			assert.Equal(t, tt.expected, resp.Code)
		})
	}
}

func float64Ptr(v float64) *float64 { return &v }
