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

	}

	if m.ID == "" || (m.MType != "gauge" && m.MType != "counter") {
		http.Error(w, "invalid metric data", http.StatusBadRequest)

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)

}

func (h *Handler)GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	var m models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}


	json, err := h.storage.GetMetricsByTypeAndName(m.ID,m.MType)
	
	if err != nil{
		http.Error(w, "metric not found", http.StatusNotFound)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)

}