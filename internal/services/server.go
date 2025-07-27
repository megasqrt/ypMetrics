package services

import (
	"net/http"
	"os"

	"ypMetrics/internal/store"
	"ypMetrics/internal/misc"
	"ypMetrics/internal/services/middlewares"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func NewMetricServer(cfg misc.Config, s store.Storage) *http.Server {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	handlers := NewHandler(s)

	router := mux.NewRouter()
	HashMiddleware:=middlewares.NewHashMiddleware(cfg.HashKey)
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
