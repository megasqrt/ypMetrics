package services

import (
	"ypMetrics/internal/misc"
	"ypMetrics/internal/services/middlewares"
	"ypMetrics/internal/store"

	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

func NewMetricServer(cfg misc.Config, s store.Storage, log zerolog.Logger) *http.Server {
	metricService := NewMetricService(s, log)
	handlers := NewHandler(metricService, log)

	router := mux.NewRouter()
	HashMiddleware := middlewares.NewHashMiddleware(cfg.HashKey)
	router.Use(middlewares.LoggingMiddleware, HashMiddleware.HashMiddleware, middlewares.GzipMiddleware)

	router.HandleFunc("/update/", handlers.UpdateMetricJSON).Methods(http.MethodPost)
	router.HandleFunc("/updates/", handlers.UpdateMetricsBatchJSON).Methods(http.MethodPost)
	router.HandleFunc("/value/", handlers.GetMetricJSON).Methods(http.MethodPost)

	router.HandleFunc("/update/{type}/{value}", handlers.errorHandler).Methods(http.MethodPost)
	router.HandleFunc("/update/{type}/{name}/{value}", handlers.updateHandler).Methods(http.MethodPost)

	router.HandleFunc("/value/{type}/{name}", handlers.getMetricHandler).Methods(http.MethodGet)

	router.HandleFunc("/metrics", handlers.metricsHandler).Methods(http.MethodPost)

	router.HandleFunc("/", handlers.metricsHTMLHandler).Methods(http.MethodGet)

	router.HandleFunc("/ping", handlers.dbPingHandler).Methods(http.MethodGet)

	return &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: router,
	}
}
