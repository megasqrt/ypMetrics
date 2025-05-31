package services

import (
	"encoding/json"
	"fmt"
	"ypMetrics/internal/metrics"
	"ypMetrics/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type MetricsHandler struct{}

// MetricServer представляет HTTP сервер для работы с метриками
type MetricServer struct {
	storage metrics.MemStorage
}

// NewMetricServer создает новый сервер метрик
func NewMetricServer(storage *metrics.MemStorage) *MetricServer {
	server := &MetricServer{storage: *storage}

	router := mux.NewRouter()
	fmt.Println("Starting server on :8080")

	router.HandleFunc("/update/{type}/{value}", server.errorHandler).Methods(http.MethodPost)
	router.HandleFunc("/update/{type}/{name}/{value}", server.updateHandler).Methods(http.MethodPost)

	router.HandleFunc("/metrics", server.metricsHandler).Methods(http.MethodPost)
	if err := http.ListenAndServe(":8080", router); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
	return server
}

func (s *MetricServer) updateHandler(w http.ResponseWriter, r *http.Request) {
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
		s.storage.UpdateGauge(metricName, value)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Gauge %s updated to %f", metricName, value)
	case models.Counter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Invalid counter value", http.StatusBadRequest)
			return
		}
		newValue := s.storage.UpdateCounter(metricName, value)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Counter %s incremented by %d, new value: %d", metricName, value, newValue)
	default:
		mes := fmt.Sprintf("Invalid metric type", metricType)
		http.Error(w, mes, http.StatusBadRequest)
	}
}

func (s *MetricServer) errorHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Invalid URL format", http.StatusNotFound)
}

func (s *MetricServer) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := s.storage.GetAllMetrics()
	jsonData, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		http.Error(w, "Failed to serialize metrics", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}
