package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"ypMetrics/internal/store"
	"ypMetrics/models"

	"github.com/rs/zerolog"
)

// MetricService инкапсулирует бизнес-логику для работы с метриками.
type MetricService struct {
	storage store.Storage
	log     zerolog.Logger
}

// NewMetricService создает новый экземпляр MetricService.
func NewMetricService(s store.Storage, log zerolog.Logger) *MetricService {
	return &MetricService{
		storage: s,
		log:     log,
	}
}

// UpdateMetricFromURL обновляет метрику на основе данных из URL.
func (s *MetricService) UpdateMetricFromURL(metricType, metricName, metricValue string) error {
	if metricName == "" {
		return fmt.Errorf("metric name is required")
	}

	switch metricType {
	case models.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return fmt.Errorf("invalid gauge value: %w", err)
		}
		s.storage.UpdateGauge(metricName, value)
		s.log.Info().Str("name", metricName).Float64("value", value).Msg("Gauge updated")
	case models.Counter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid counter value: %w", err)
		}
		s.storage.UpdateCounter(metricName, value)
		s.log.Info().Str("name", metricName).Int64("delta", value).Msg("Counter updated")
	default:
		err := fmt.Errorf("invalid metric type %s", metricType)
		s.log.Error().Err(err).Msg("update metric from url")
		return err
	}
	return nil
}

// UpdateMetricJSON обновляет метрику из JSON-объекта.
func (s *MetricService) UpdateMetricJSON(m models.Metrics) (models.Metrics, error) {
	if m.ID == "" || (m.MType != models.Gauge && m.MType != models.Counter) {
		s.log.Error().Str("type", m.MType).Msg("invalid metric data type")
		return models.Metrics{}, fmt.Errorf("invalid metric data")
	}

	s.log.Info().Str("type", m.MType).Str("id", m.ID).Msg("Updating metric")

	switch m.MType {
	case models.Gauge:
		if m.Value == nil {
			return models.Metrics{}, fmt.Errorf("value required for gauge")
		}
		s.storage.UpdateGauge(m.ID, *m.Value)
	case models.Counter:
		if m.Delta == nil {
			return models.Metrics{}, fmt.Errorf("delta required for counter")
		}
		s.storage.UpdateCounter(m.ID, *m.Delta)
	}

	updatedMetricJSON, err := s.storage.GetJSONMetricsByTypeAndName(m.ID, m.MType)
	if err != nil {
		s.log.Error().Err(err).Msg("could not retrieve updated metric")
		return models.Metrics{}, fmt.Errorf("could not retrieve updated metric: %w", err)
	}

	var updatedMetric models.Metrics
	if err := json.Unmarshal(updatedMetricJSON, &updatedMetric); err != nil {
		s.log.Error().Err(err).Msg("could not unmarshal updated metric")
		return models.Metrics{}, fmt.Errorf("could not unmarshal updated metric: %w", err)
	}

	return updatedMetric, nil
}

// GetMetricValue возвращает значение метрики.
func (s *MetricService) GetMetricValue(metricType, metricName string) ([]byte, error) {
	return s.storage.GetMetricsByTypeAndName(metricName, metricType)
}

// GetMetricJSON возвращает метрику в формате JSON.
func (s *MetricService) GetMetricJSON(m models.Metrics) ([]byte, error) {
	return s.storage.GetJSONMetricsByTypeAndName(m.ID, m.MType)
}

// GetAllMetrics возвращает все метрики.
func (s *MetricService) GetAllMetrics() map[string]interface{} {
	return s.storage.GetAllMetrics()
}

// UpdateMetricsBatch обновляет метрики пакетом.
func (s *MetricService) UpdateMetricsBatch(metrics []models.Metrics) error {
	return s.storage.UpdateMetricsBatch(metrics)
}

// Ping проверяет доступность хранилища.
func (s *MetricService) Ping(ctx context.Context) error {
	return s.storage.Ping(ctx)
}
