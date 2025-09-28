package metrics

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetMetric(t *testing.T) {
	storage := NewMemStorage(nil, 0)
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
			jsonData, err := storage.GetMetricsByTypeAndName(tt.metricName, tt.metricType)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)

			switch v := tt.wantValue.(type) {
			case float64:
				var gotValue float64
				err := json.Unmarshal(jsonData, &gotValue)
				require.NoError(t, err)
				assert.InDelta(t, v, gotValue, 0.001)
			case int64:
				var gotValue int64
				err := json.Unmarshal(jsonData, &gotValue)
				require.NoError(t, err)
				assert.Equal(t, v, gotValue)
			}
		})
	}
}