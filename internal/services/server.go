package services

import (
	"net/http"
	"os"

	"ypMetrics/internal/store"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func NewMetricServer(address string, s store.Storage) *http.Server {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	handlers := &Handler{storage: s}

	router := mux.NewRouter()
	router.Use(LoggingMiddleware,GzipMiddleware)


	router.HandleFunc("/update/", handlers.UpdateMetricJSON).Methods(http.MethodPost)
    router.HandleFunc("/value/", handlers.GetMetricJSON).Methods(http.MethodPost)

	router.HandleFunc("/update/{type}/{value}", handlers.errorHandler).Methods(http.MethodPost)
	router.HandleFunc("/update/{type}/{name}/{value}", handlers.updateHandler).Methods(http.MethodPost)
	
	router.HandleFunc("/value/{type}/{name}", handlers.getMetricHandler).Methods(http.MethodGet)

	router.HandleFunc("/metrics", handlers.metricsHandler).Methods(http.MethodPost)
	
	router.HandleFunc("/", handlers.metricsHTMLHandler).Methods(http.MethodGet)

	return &http.Server{
		Addr:    address,
		Handler: router,
	}
}
