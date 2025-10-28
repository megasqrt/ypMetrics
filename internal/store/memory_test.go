package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
	"ypMetrics/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test-metrics.json")
	if content != "" {
		err := os.WriteFile(filePath, []byte(content), 0666)
		require.NoError(t, err)
	}
	return filePath
}

func TestNewMemStorage(t *testing.T) {
	t.Run("create without restore", func(t *testing.T) {
		s, err := NewMemStorage("", 0, false)
		require.NoError(t, err)
		assert.NotNil(t, s)
		assert.Empty(t, s.gauges)
		assert.Empty(t, s.counters)
	})

	t.Run("create with restore from non-existent file", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "non-existent.json")
		s, err := NewMemStorage(filePath, 0, true)
		require.NoError(t, err)
		assert.NotNil(t, s)
		assert.Empty(t, s.gauges)
		assert.Empty(t, s.counters)
	})

	t.Run("create with restore from existing file", func(t *testing.T) {
		content := `{"gauges":{"g1":1.23},"counters":{"c1":123}}`
		filePath := createTempFile(t, content)

		s, err := NewMemStorage(filePath, 0, true)
		require.NoError(t, err)
		assert.NotNil(t, s)
		assert.Equal(t, 1.23, s.gauges["g1"])
		assert.Equal(t, int64(123), s.counters["c1"])
	})

	t.Run("create with restore from corrupted file", func(t *testing.T) {
		filePath := createTempFile(t, `{"gauges":corrupted}`)
		_, err := NewMemStorage(filePath, 0, true)
		require.NoError(t, err) 
	})
}

func TestMemStorage_UpdateAndGet(t *testing.T) {
	s, _ := NewMemStorage("", 0, false)
	ctx := context.Background()

	s.UpdateGauge(ctx, "g1", 10.5)
	newValue := s.UpdateCounter(ctx, "c1", 5)

	assert.Equal(t, int64(5), newValue)
	assert.Equal(t, 10.5, s.gauges["g1"])
	assert.Equal(t, int64(5), s.counters["c1"])

	newValue = s.UpdateCounter(ctx, "c1", 10)
	assert.Equal(t, int64(15), newValue)
	assert.Equal(t, int64(15), s.counters["c1"])

	allMetrics := s.GetAllMetrics()
	expectedGauges := map[string]float64{"g1": 10.5}
	expectedCounters := map[string]int64{"c1": 15}
	assert.Equal(t, expectedGauges, allMetrics["gauges"])
	assert.Equal(t, expectedCounters, allMetrics["counters"])

	gVal, err := s.GetMetricsByTypeAndName("g1", models.Gauge)
	require.NoError(t, err)
	assert.Equal(t, "10.5", string(gVal))

	cVal, err := s.GetMetricsByTypeAndName("c1", models.Counter)
	require.NoError(t, err)
	assert.Equal(t, "15", string(cVal))

	_, err = s.GetMetricsByTypeAndName("non-existent", models.Gauge)
	assert.Error(t, err)

	gJSON, err := s.GetJSONMetricsByTypeAndName("g1", models.Gauge)
	require.NoError(t, err)
	var gMetric models.Metrics
	err = json.Unmarshal(gJSON, &gMetric)
	require.NoError(t, err)
	assert.Equal(t, "g1", gMetric.ID)
	assert.Equal(t, models.Gauge, gMetric.MType)
	assert.Equal(t, 10.5, *gMetric.Value)

	cJSON, err := s.GetJSONMetricsByTypeAndName("c1", models.Counter)
	require.NoError(t, err)
	var cMetric models.Metrics
	err = json.Unmarshal(cJSON, &cMetric)
	require.NoError(t, err)
	assert.Equal(t, "c1", cMetric.ID)
	assert.Equal(t, models.Counter, cMetric.MType)
	assert.Equal(t, int64(15), *cMetric.Delta)

	_, err = s.GetJSONMetricsByTypeAndName("non-existent", models.Counter)
	assert.Error(t, err)
}

func TestMemStorage_UpdateMetricsBatch(t *testing.T) {
	s, _ := NewMemStorage("", 0, false)
	ctx := context.Background()

	metrics := []models.Metrics{
		{ID: "g1", MType: models.Gauge, Value: float64Ptr(1.1)},
		{ID: "c1", MType: models.Counter, Delta: int64Ptr(10)},
		{ID: "g1", MType: models.Gauge, Value: float64Ptr(2.2)}, // Overwrite g1
		{ID: "c1", MType: models.Counter, Delta: int64Ptr(5)},   // Increment c1
	}

	err := s.UpdateMetricsBatch(ctx, metrics)
	require.NoError(t, err)

	assert.Equal(t, 2.2, s.gauges["g1"])
	assert.Equal(t, int64(15), s.counters["c1"])
}

func TestMemStorage_FileOperations(t *testing.T) {
	t.Run("sync save", func(t *testing.T) {
		filePath := createTempFile(t, "")
		s, err := NewMemStorage(filePath, 0, false)
		require.NoError(t, err)

		s.UpdateGauge(context.Background(), "g_sync", 99.9)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)

		var result map[string]map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		assert.Equal(t, 99.9, result["gauges"]["g_sync"])
	})

	t.Run("periodic save and save on exit", func(t *testing.T) {
		filePath := createTempFile(t, "")
		s, err := NewMemStorage(filePath, 100*time.Millisecond, false)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s.StartPeriodicSave(ctx)

		s.UpdateGauge(ctx, "g_periodic", 1.23)
		time.Sleep(150 * time.Millisecond) 

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"g_periodic":1.23`)

		s.UpdateCounter(ctx, "c_exit", 789)
		s.SaveOnExit()

		data, err = os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"c_exit":789`)
	})

	t.Run("load from file", func(t *testing.T) {
		content := `{"gauges":{"g_load":5.5},"counters":{"c_load":55}}`
		filePath := createTempFile(t, content)
		s, err := NewMemStorage(filePath, 0, false)
		require.NoError(t, err)

		err = s.loadFromFile()
		require.NoError(t, err)

		assert.Equal(t, 5.5, s.gauges["g_load"])
		assert.Equal(t, int64(55), s.counters["c_load"])
	})
}

func TestMemStorage_Ping(t *testing.T) {
	s, _ := NewMemStorage("", 0, false)
	err := s.Ping(context.Background())
	assert.NoError(t, err)
}

func TestMemStorage_GetAllMetrics_ReturnsCopy(t *testing.T) {
	s, _ := NewMemStorage("", 0, false)
	s.UpdateGauge(context.Background(), "g1", 1.0)

	allMetrics := s.GetAllMetrics()
	gauges := allMetrics["gauges"].(map[string]float64)

	gauges["g1"] = 2.0

	assert.Equal(t, 1.0, s.gauges["g1"])
}

func float64Ptr(f float64) *float64 {
	return &f
}

func int64Ptr(i int64) *int64 {
	return &i
}
