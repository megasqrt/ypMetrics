package services

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"ypMetrics/internal/helper"
	"ypMetrics/internal/store"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

)

func NewMetricServer(storage store.Storage) error{
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	viper.AutomaticEnv() 
    var serverAddress string
    envAddress := viper.GetString("ADDRESS") 
	flag.StringVar(&serverAddress, "a", "localhost:8080", "server adress")

	flag.Parse()

	helper.AssignIfNotEmpty(&serverAddress,envAddress)

	handlers := &Handler{storage: storage}

	router := mux.NewRouter()
	fmt.Printf("Starting server on %s\n",serverAddress)
	router.Use(LoggingMiddleware)

	router.HandleFunc("/update/", handlers.UpdateMetricJSON).Methods(http.MethodPost)
    router.HandleFunc("/value/", handlers.GetMetricJSON).Methods(http.MethodPost)

	router.HandleFunc("/update/{type}/{value}", handlers.errorHandler).Methods(http.MethodPost)
	router.HandleFunc("/update/{type}/{name}/{value}", handlers.updateHandler).Methods(http.MethodPost)
	
	router.HandleFunc("/value/{type}/{name}", handlers.getMetricHandler).Methods(http.MethodGet)

	router.HandleFunc("/metrics", handlers.metricsHandler).Methods(http.MethodPost)
	
	router.HandleFunc("/", handlers.metricsHTMLHandler).Methods(http.MethodGet)

	if err := http.ListenAndServe(serverAddress, router); err != nil {
		return fmt.Errorf("server error: %v", err)
	}
	return nil
}
