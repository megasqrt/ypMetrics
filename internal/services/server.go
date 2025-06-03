package services

import (
	
	"fmt"
	"ypMetrics/internal/metrics"
	"net/http"
	"github.com/gorilla/mux"
)

func NewMetricServer(storage *metrics.MemStorage) {
	handlers := &Handler{storage: *storage}

	router := mux.NewRouter()
	fmt.Println("Starting server on :8080")

	router.HandleFunc("/update/{type}/{value}", handlers.errorHandler).Methods(http.MethodPost)
	router.HandleFunc("/update/{type}/{name}/{value}", handlers.updateHandler).Methods(http.MethodPost)
	
	router.HandleFunc("/value/{type}/{name}", handlers.getMetricHandler).Methods(http.MethodGet)

	router.HandleFunc("/metrics", handlers.metricsHandler).Methods(http.MethodPost)
	
	router.HandleFunc("/", handlers.metricsHtmlHandler).Methods(http.MethodGet)


	if err := http.ListenAndServe(":8080", router); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
