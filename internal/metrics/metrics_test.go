package metrics

import (
	"encoding/json"
	"testing"
	"strconv"
	"unsafe"
	"github.com/stretchr/testify/assert"
)

func TestGetMetric(t *testing.T) {
	storage := NewMemStorage()
	storage.UpdateGauge("temperature", 36.6)
	storage.UpdateCounter("requests", 42)

	tests := []struct {
		name        string
		metricType  string
		metricName  string
		wantValue   interface{}
		wantErr     bool
		errContains string
	}{
		{
			name:       "existing gauge",
			metricType: "gauge",
			metricName: "temperature",
			wantValue:  36.6,
		},
		{
			name:       "existing counter",
			metricType: "counter",
			metricName: "requests",
			wantValue:  int64(42),
		},
		{
			name:        "nonexistent gauge",
			metricType:  "gauge",
			metricName:  "humidity",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:        "invalid type",
			metricType:  "invalid",
			metricName:  "temperature",
			wantErr:     true,
			errContains: "nvalid metric type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := storage.GetMetricsByTypeAndName( tt.metricName,tt.metricType)
			
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}
			
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}


			err = json.Unmarshal(jsonData, &tt.wantValue)
			
			assert.NoError(t, err, "Unmarshal should not fail")
			assert.Equal(t, tt.wantValue, bytesToFloat64Fast(jsonData), "Values should be equal")
			
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func bytesToFloat64Fast(b []byte) (float64) {
	value,_:=strconv.ParseFloat(unsafe.String(unsafe.SliceData(b), len(b)), 64)
    return value
}