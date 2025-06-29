package services

import (
    "encoding/json"
    "net/http"
    "ypMetrics/models"
)

func (h *Handler)UpdateMetricJSON(w http.ResponseWriter, r *http.Request){

	var m models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if m.ID == "" || (m.MType != "gauge" && m.MType != "counter") {
		http.Error(w, "invalid metric data", http.StatusBadRequest)
		return
	}

	switch m.MType {
	case "gauge":
		if m.Value == nil {
			http.Error(w, "value required for gauge", http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(m.ID, *m.Value)
	case "counter":
		if m.Delta == nil {
			http.Error(w, "delta required for counter", http.StatusBadRequest)
			return
		}
		h.storage.UpdateCounter(m.ID, *m.Delta)
	}

	updatedMetric, err := h.storage.GetJSONMetricsByTypeAndName(m.ID, m.MType)
	if err != nil {
		http.Error(w, "could not retrieve updated metric", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(updatedMetric)
}

func (h *Handler)GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	var m models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	metricJSON, err := h.storage.GetJSONMetricsByTypeAndName(m.ID, m.MType)
	
	if err != nil {
		http.Error(w, "metric not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(metricJSON)
}