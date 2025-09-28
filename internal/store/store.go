package store

import (
	"context"
	"ypMetrics/models"
)

type Storage interface {
	UpdateGauge(name string, value float64) 
	UpdateCounter(name string, value int64) int64 
	GetAllMetrics() map[string] interface{}
	GetMetricsByTypeAndName(mName, mType string) ([]byte, error) 
	GetJSONMetricsByTypeAndName(mName, mType string) ([]byte, error)

	UpdateMetricsBatch(metrics []models.Metrics) error
	Ping(ctx context.Context) error
}