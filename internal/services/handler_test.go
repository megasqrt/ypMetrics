package services

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ypMetrics/internal/mocks"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)


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
			mockStorage := &mocks.MockStorage{}
			handler := NewHandler(mockStorage)
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
	
	mockStorage := &mocks.MockStorage{}
	mockStorage.WithGauge( "temperature", 36.6).
				WithCounter("requests", 42)
    
	handler := NewHandler(mockStorage)


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

