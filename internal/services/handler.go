package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ypMetrics/models"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

type Handler struct {
	service *MetricService
	log     zerolog.Logger
}

// NewHandler создает новый экземпляр Handler с предоставленным хранилищем.
func NewHandler(service *MetricService, log zerolog.Logger) Handler {
	return Handler{
		service: service,
		log:     log,
	}
}

// updateHandler обрабатывает запросы на обновление метрик через URL.
// Принимает метрики в формате /update/{type}/{name}/{value}.
//
//   - {type} — тип метрики (gauge или counter).
//   - {name} — имя метрики.
//   - {value} — значение метрики.
func (h *Handler) updateHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	metricType := vars["type"]
	metricName := vars["name"]
	metricValue := vars["value"]

	err := h.service.UpdateMetricFromURL(metricType, metricName, metricValue)
	if err != nil {
		// Определяем код ошибки в зависимости от типа
		if metricName == "" {
			http.Error(w, "Invalid URL format", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// errorHandler обрабатывает некорректные URL, возвращая ошибку 404 Not Found.
func (h *Handler) errorHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Invalid URL format", http.StatusNotFound)
}

// metricsHandler возвращает все текущие метрики в формате JSON.
// Используется для отладки и мониторинга.
func (h *Handler) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := h.service.GetAllMetrics()
	jsonData, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		http.Error(w, "Failed to serialize metrics", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// metricsHTMLHandler отображает все метрики в виде HTML-страницы.
// Предназначен для удобного просмотра метрик в браузере.
func (h *Handler) metricsHTMLHandler(w http.ResponseWriter, r *http.Request) {

	metrics := h.service.GetAllMetrics()

	html := models.HTMLHead

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

	fmt.Fprint(w, html)
}

func (h *Handler) getMetricHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	metricType := vars["type"]
	metricName := vars["name"]

	if metricType == "" || metricName == "" {
		http.Error(w, "Invalid URL format", http.StatusNotFound)
		return
	}

	jsonData, err := h.service.GetMetricValue(metricType, metricName)
	if err != nil {
		h.log.Warn().Err(err).Str("name", metricName).Str("type", metricType).Msg("Metric not found")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Metric '%s' of type '%s' not found", metricName, metricType)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
	}
}

// dbPingHandler проверяет соединение с базой данных.
// Возвращает статус 200 OK, если соединение успешно,
// и 500 Internal Server Error в противном случае.
func (h *Handler) dbPingHandler(w http.ResponseWriter, r *http.Request) {

	if err := h.service.Ping(r.Context()); err != nil {
		h.log.Error().Err(err).Msg("Storage ping failed")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// UpdateMetricsBatchJSON обрабатывает пакетное обновление метрик.
// Принимает в теле запроса JSON-массив метрик в формате []models.Metrics.
// В случае успеха возвращает статус 200 OK.
func (h *Handler) UpdateMetricsBatchJSON(w http.ResponseWriter, r *http.Request) {
	var metrics []models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateMetricsBatch(metrics); err != nil {
		http.Error(w, "Failed to update metrics batch", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
