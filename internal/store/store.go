package store

import (
	"context"
	"ypMetrics/models"
)

// Storage определяет интерфейс для взаимодействия с хранилищем метрик.
// Он абстрагирует реализацию хранения, позволяя использовать как in-memory,
// так и персистентные хранилища (например, PostgreSQL).
type Storage interface {
	// UpdateGauge обновляет или создает метрику типа gauge.
	UpdateGauge(name string, value float64)
	// UpdateCounter обновляет или создает метрику типа counter и возвращает ее новое значение.
	UpdateCounter(name string, value int64) int64
	// GetAllMetrics возвращает все метрики из хранилища.
	GetAllMetrics() map[string]interface{}
	// GetMetricsByTypeAndName возвращает значение метрики по ее типу и имени в виде среза байт.
	GetMetricsByTypeAndName(mName, mType string) ([]byte, error)
	// GetJSONMetricsByTypeAndName возвращает метрику в виде JSON-объекта models.Metrics.
	GetJSONMetricsByTypeAndName(mName, mType string) ([]byte, error)
	// UpdateMetricsBatch выполняет пакетное обновление метрик.
	UpdateMetricsBatch(metrics []models.Metrics) error
	// Ping проверяет доступность хранилища.
	Ping(ctx context.Context) error
}
