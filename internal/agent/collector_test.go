package agent

import (
	"testing"

	"ypMetrics/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricCollector(t *testing.T) {
	c := NewMetricCollector()
	require.NotNil(t, c)
	assert.NotNil(t, c.metrics)
	assert.Empty(t, c.metrics)
	assert.Equal(t, int64(0), c.pollCount)
}

func TestMetricCollector_Poll(t *testing.T) {
	c := NewMetricCollector()

	c.Poll()

	metrics := c.GetMetrics()
	assert.Len(t, metrics, 29)

	metricsMap := toMap(metrics)

	pollCount, ok := metricsMap["PollCount"]
	require.True(t, ok, "PollCount metric should exist")
	assert.Equal(t, models.Counter, pollCount.MType)
	require.NotNil(t, pollCount.Delta)
	assert.Equal(t, int64(1), *pollCount.Delta)

	c.Poll()
	metrics2 := c.GetMetrics()
	assert.Len(t, metrics2, 29)

	metricsMap2 := toMap(metrics2)
	pollCount2, ok := metricsMap2["PollCount"]
	require.True(t, ok)
	require.NotNil(t, pollCount2.Delta)
	assert.Equal(t, int64(2), *pollCount2.Delta, "PollCount should be incremented")
}

func toMap(metrics []models.Metrics) map[string]models.Metrics {
	m := make(map[string]models.Metrics, len(metrics))
	for _, metric := range metrics {
		m[metric.ID] = metric
	}
	return m
}
