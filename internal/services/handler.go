package services

import (
	"ypMetrics/internal/metrics"
	"encoding/json"
	"ypMetrics/models"
	"strconv"
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
)

type Handler struct {
	storage metrics.MemStorage
}

func NewHandler() Handler {
	var h Handler
	h.storage = *metrics.NewMemStorage()
	return h
}

func (h *Handler) updateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	metricType := vars["type"]
	metricName := vars["name"]
	metricValue := vars["value"]

	if metricName == "" {
		http.Error(w, "Invalid URL format", http.StatusNotFound)
		return
	}

	switch metricType {
	case models.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Invalid gauge value", http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(metricName, value)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Gauge %s updated to %f", metricName, value)
	case models.Counter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Invalid counter value", http.StatusBadRequest)
			return
		}
		newValue := h.storage.UpdateCounter(metricName, value)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Counter %s incremented by %d, new value: %d", metricName, value, newValue)
	default:
		mes := fmt.Sprintf("Invalid metric type %s", metricType)
		http.Error(w, mes, http.StatusBadRequest)
	}
}

func (h *Handler) errorHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Invalid URL format", http.StatusNotFound)
}

func (h *Handler) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := h.storage.GetAllMetrics()
	jsonData, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		http.Error(w, "Failed to serialize metrics", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}
