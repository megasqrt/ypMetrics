package services

import (
	"encoding/json"
	"net/http"
	"ypMetrics/internal/helper"
	"ypMetrics/models"
)

// UpdateMetricJSON обрабатывает обновление одной метрики в формате JSON.
// Принимает в теле запроса JSON-объект models.Metrics.
//
// Проверяет корректность типа метрики ("gauge" или "counter") и наличие
// соответствующего поля (Value или Delta).
//
// В случае успеха обновляет метрику в хранилище и возвращает
// обновленный JSON-объект метрики с кодом 200 OK.
func (h *Handler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {

	var m models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		h.log.Error().Err(err).Msg("Error update decoding JSON")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updatedMetric, err := h.service.UpdateMetricJSON(m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest) // Может быть и 500, но сервис вернет ошибку
		return
	}

	resp, err := json.Marshal(updatedMetric)
	if err != nil {
		http.Error(w, "could not marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

// GetMetricJSON возвращает метрику в формате JSON.
// Принимает в теле запроса JSON-объект с полями "id" и "type".
//
// В случае успеха возвращает полный JSON-объект models.Metrics с кодом 200 OK.
// Если метрика не найдена, возвращает 404 Not Found.
func (h *Handler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	var m models.Metrics
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		h.log.Error().Err(err).Msg("Error decoding JSON")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	metricJSON, err := h.service.GetMetricJSON(m)
	if err != nil {
		h.log.Warn().Err(err).Msg("json metric error")
		helper.JSONErrorWithMesage(w, http.StatusNotFound, "json metric not found")
		return
	}

	w.Write(metricJSON)
}
