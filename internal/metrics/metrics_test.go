package metrics

import (
	"encoding/json"
	"testing"
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

			var result struct {
				Value interface{} `json:"value"`
			}
			
			if err := json.Unmarshal(jsonData, &result); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			switch v := tt.wantValue.(type) {
			case float64:
				if val, ok := result.Value.(float64); !ok || val != v {
					t.Errorf("expected value %v, got %v", v, result.Value)
				}
			case int64:
				if val, ok := result.Value.(float64); !ok || int64(val) != v {
					t.Errorf("expected value %v, got %v", v, result.Value)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}