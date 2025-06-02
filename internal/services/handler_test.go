package services

import (
	"net/http"
	"net/http/httptest"
	"testing"

	//"github.com/golangci/golangci-lint/pkg/golinters/bodyclose"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	//"github.com/stretchr/testify/require"
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
				body: `{"status":"ok"}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler()
			request,_ := http.NewRequest(http.MethodPost, "/update", nil)
			record := httptest.NewRecorder()
			vars := map[string]string{
				"type":  tt.want.mType,
				"name":  tt.want.mName,
				"value": tt.want.mValue,
			}
			request = mux.SetURLVars(request, vars)
            handler.updateHandler(record, request)
			assert.Equal(t, http.StatusOK, record.Code)
			assert.Equal(t, tt.want.body, record.Body.String())
			// result := record.Result()
			
			// assert.Equal(t, tt.want.statusCode, result.StatusCode)
            // assert.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))

			
		})
	}
}
