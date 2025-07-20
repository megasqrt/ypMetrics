package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/spf13/viper"

	"ypMetrics/internal/agent"
	"ypMetrics/internal/helper"
)

type MetricsAgent struct {
	collector      agent.Collector
	reporter       agent.Reporter
	pollInterval   time.Duration
	reportInterval time.Duration
}

func NewMetricsAgent(collector agent.Collector, reporter agent.Reporter, pollInterval, reportInterval time.Duration) *MetricsAgent {
	return &MetricsAgent{
		collector:      collector,
		reporter:       reporter,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
	}
}

func (a *MetricsAgent) Run(ctx context.Context) {
	pollTicker := time.NewTicker(a.pollInterval)
	defer pollTicker.Stop()
	reportTicker := time.NewTicker(a.reportInterval)
	defer reportTicker.Stop()

	log.Println("Агент запущен.")

	// Первоначальный сбор метрик, чтобы были данные для первой отправки.
	a.collector.Poll()

	for {
		select {
		case <-pollTicker.C:
			a.collector.Poll()
		case <-reportTicker.C:
			metrics := a.collector.GetMetrics()
			if err := a.reporter.Report(metrics); err != nil {
				log.Printf("Не удалось отправить метрики: %v", err)
			}
		case <-ctx.Done():
			log.Println("Агент останавливается. Отправка финального отчета...")
			metrics := a.collector.GetMetrics()
			if err := a.reporter.Report(metrics); err != nil {
				log.Printf("Не удалось отправить финальный отчет: %v", err)
			}
			log.Println("Агент остановлен.")
			return
		}
	}
}

var (
	serverAddress  string
	reportInterval int
	pollInterval   int
	hashKey        string
)

type config struct {
	serverAddress  string
	pollInterval   time.Duration
	reportInterval time.Duration
	hashKey        string
}

const (
	defaultServerAddress  = "localhost:8080"
	defaultReportInterval = 10
	defaultPollInterval   = 2
	defaultHashKey        = ""
)

func registerFlags() {
	flag.StringVar(&serverAddress, "a", defaultServerAddress, "server adress")
	flag.IntVar(&reportInterval, "r", defaultReportInterval, "report interval")
	flag.IntVar(&pollInterval, "p", defaultPollInterval, "poll interval")
	flag.StringVar(&hashKey, "k", defaultHashKey, "key for hashing")
}

func init() {
	registerFlags()
}

func parseConfig() config {
	flag.Parse()

	viper.AutomaticEnv()

	helper.AssignFromViperIfSet(&serverAddress, "ADDRESS", viper.GetString, defaultServerAddress)
	helper.AssignFromViperIfSet(&reportInterval, "REPORT_INTERVAL", viper.GetInt, defaultReportInterval)
	helper.AssignFromViperIfSet(&pollInterval, "POLL_INTERVAL", viper.GetInt, defaultPollInterval)
	helper.AssignFromViperIfSet(&hashKey, "KEY", viper.GetString, defaultHashKey)
	
	if !govalidator.IsURL(serverAddress) {
		log.Fatalf("некорректный URL сервера: %s", serverAddress)
	}

	return config{
		serverAddress:  serverAddress,
		reportInterval: time.Duration(reportInterval) * time.Second,
		pollInterval:   time.Duration(pollInterval) * time.Second,
		hashKey:        hashKey,
	}
}

func main() {
	cfg := parseConfig()

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Создание компонентов
	collector := agent.NewMetricCollector()
	reporter := agent.NewHTTPReporter(cfg.serverAddress, cfg.hashKey)

	// Создание и запуск агента
	agent := NewMetricsAgent(
		collector,
		reporter,
		cfg.pollInterval,
		cfg.reportInterval,
	)
	go agent.Run(ctx)

	sig := <-sigChan
	log.Printf("Получен сигнал: %v. Завершение работы...", sig)
	cancel()
	time.Sleep(time.Second) // Ждем завершения горутин
}
