package services

import (
	
	"fmt"
	"ypMetrics/internal/metrics"
	"net/http"
	"github.com/gorilla/mux"
	"flag"
	"github.com/spf13/viper"
)

func NewMetricServer(storage *metrics.MemStorage) {
	viper.AutomaticEnv() 
    var serverAddress string
    envAddress := viper.GetString("ADDRESS") 
	flag.StringVar(&serverAddress, "a", "localhost:8080", "server adress")

	flag.Parse()

	if envAddress != "" {
		serverAddress = envAddress
	}

	handlers := &Handler{storage: *storage}

	router := mux.NewRouter()
	fmt.Printf("Starting server on %s\n",serverAddress)

	router.HandleFunc("/update/{type}/{value}", handlers.errorHandler).Methods(http.MethodPost)
	router.HandleFunc("/update/{type}/{name}/{value}", handlers.updateHandler).Methods(http.MethodPost)
	
	router.HandleFunc("/value/{type}/{name}", handlers.getMetricHandler).Methods(http.MethodGet)

	router.HandleFunc("/metrics", handlers.metricsHandler).Methods(http.MethodPost)
	
	router.HandleFunc("/", handlers.metricsHTMLHandler).Methods(http.MethodGet)


	if err := http.ListenAndServe(serverAddress, router); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
