package store

import (
	"encoding/json"
	"os"
	"ypMetrics/internal/metrics"
)

type FileStorage struct {
	filePath string
}

func NewFileStorage(filePath string) *FileStorage {
	return &FileStorage{filePath: filePath}
}

func (fs *FileStorage) SaveMetrics(storage *metrics.MemStorage) error {
	data, err := json.Marshal(storage.GetAllMetrics())
	if err != nil {
		return err
	}

	return os.WriteFile(fs.filePath, data, 0666)
}

func (fs *FileStorage) LoadMetrics(storage *metrics.MemStorage) error {
	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var allMetrics struct {
		Gauges   map[string]float64 `json:"gauges"`
		Counters map[string]int64   `json:"counters"`
	}

	if err := json.Unmarshal(data, &allMetrics); err != nil {
		return err
	}

	for name, value := range allMetrics.Gauges {
		storage.UpdateGauge(name, value)
	}
	for name, value := range allMetrics.Counters {
		storage.UpdateCounter(name, value)
	}

	return nil
}