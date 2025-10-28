package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStorage_SaveMetrics(t *testing.T) {
	filePath := createTempFile(t, "")
	fs := &FileStorage{filePath: filePath}

	storage, err := NewMemStorage("", 0, false)
	require.NoError(t, err)

	storage.UpdateGauge(context.Background(), "g1", 123.45)
	storage.UpdateCounter(context.Background(), "c1", 100)

	err = fs.SaveMetrics(context.Background(), storage)
	require.NoError(t, err)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	expectedJSON := `{"counters":{"c1":100},"gauges":{"g1":123.45}}`
	assert.JSONEq(t, expectedJSON, string(data))
}

func TestFileStorage_LoadMetrics(t *testing.T) {
	t.Run("successful load", func(t *testing.T) {
		content := `{"gauges":{"g1":54.321},"counters":{"c1":99}}`
		filePath := createTempFile(t, content)
		fs := &FileStorage{filePath: filePath}

		storage, err := NewMemStorage("", 0, false)
		require.NoError(t, err)

		err = fs.LoadMetrics(storage)
		require.NoError(t, err)

		assert.Equal(t, 54.321, storage.gauges["g1"])
		assert.Equal(t, int64(99), storage.counters["c1"])
	})

	t.Run("load from non-existent file", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "non-existent.json")
		fs := &FileStorage{filePath: filePath}
		storage, _ := NewMemStorage("", 0, false)

		err := fs.LoadMetrics(storage)
		require.NoError(t, err) // Ошибки быть не должно
		assert.Empty(t, storage.gauges)
		assert.Empty(t, storage.counters)
	})

	t.Run("load from empty file", func(t *testing.T) {
		filePath := createTempFile(t, "")
		fs := &FileStorage{filePath: filePath}
		storage, _ := NewMemStorage("", 0, false)

		err := fs.LoadMetrics(storage)
		require.NoError(t, err) // Ошибки быть не должно
		assert.Empty(t, storage.gauges)
		assert.Empty(t, storage.counters)
	})

	t.Run("load from corrupted file", func(t *testing.T) {
		filePath := createTempFile(t, `{"gauges":corrupted}`)
		fs := &FileStorage{filePath: filePath}
		storage, _ := NewMemStorage("", 0, false)

		err := fs.LoadMetrics(storage)
		require.Error(t, err) // Должна быть ошибка парсинга JSON
	})
}
