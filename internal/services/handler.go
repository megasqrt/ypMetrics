package services

import (
	"ypMetrics/internal/metrics"
	"encoding/json"
	"ypMetrics/models"
	"strconv"
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"io"
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

func (h *Handler) metricsHtmlHandler(w http.ResponseWriter, r *http.Request) {

    metrics := h.storage.GetAllMetrics()
    
    html:= models.HtmlHead

    if gauges, ok := metrics["gauges"].(map[string]float64); ok && len(gauges) > 0 {
        html += `<div class="metric-section">
            <h2>Gauge Metrics</h2>`
        
        for name, value := range gauges {
            html += fmt.Sprintf(`
            <div class="metric-item">
                <span class="metric-name">%s:</span>
                <span class="metric-value">%.2f</span>
            </div>`, name, value)
        }
        html += `</div>`
    }

    if counters, ok := metrics["counters"].(map[string]int64); ok && len(counters) > 0 {
        html += `<div class="metric-section">
            <h2>Counter Metrics</h2>`
        
        for name, value := range counters {
            html += fmt.Sprintf(`
            <div class="metric-item">
                <span class="metric-name">%s:</span>
                <span class="metric-value">%d</span>
            </div>`, name, value)
        }        
        html += `</div>`
    }

    if len(metrics) == 0 {
        html += `<p>No metrics available</p>`
    }

    html += `</body></html>`

    w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	
	_, err := io.WriteString(w, html)
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
}


func (h *Handler) getMetricHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	metricType := vars["type"]
	metricName := vars["name"]

	if metricName == "" || metricType == ""{
		http.Error(w, "Invalid URL format", http.StatusNotFound)
		return
	}

	jsonData, err:= h.storage.GetMetricsByTypeAndName(metricName, metricType)
	if err!=nil{
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "ERROR Handler: %s", err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
	}	
}