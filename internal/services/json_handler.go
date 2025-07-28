package services

import (
	"encoding/json"
	"log"
	"net/http"
	"ypMetrics/internal/helper"
	"ypMetrics/models"
)

func (h *Handler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {

	var m models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		log.Printf("Error update decoding JSON: %s \n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if m.ID == "" || (m.MType != "gauge" && m.MType != "counter") {
		log.Printf("invalid metric data type %s \n", m.MType)
		http.Error(w, "invalid metric data", http.StatusBadRequest)
		return
	}
	log.Printf("mType %s \n ID %s \n", m.MType, m.ID)
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
		log.Printf("could not retrieve updated metric: %s \n", err)
		http.Error(w, "could not retrieve updated metric", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(updatedMetric)
}

func (h *Handler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	var m models.Metrics
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		log.Printf("Error decoding JSON: %s \n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	metricJSON, err := h.storage.GetJSONMetricsByTypeAndName(m.ID, m.MType)
	if err != nil {
		log.Printf("json metric error: %s \n", err)
		helper.JSONErrorWithMesage(w, http.StatusNotFound, "json metric not found")
		return
	}

	w.Write(metricJSON)
}
