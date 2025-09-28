package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/spf13/viper"

	"ypMetrics/internal/agent"
	"ypMetrics/internal/helper"
	"ypMetrics/models"
)

type MetricsAgent struct {
	collector      agent.Collector
	reporter       agent.Reporter
	pollInterval   time.Duration
	reportInterval time.Duration
	rateLimit      int
}

func NewMetricsAgent(collector agent.Collector, reporter agent.Reporter, pollInterval, reportInterval time.Duration, rateLimit int) *MetricsAgent {
	return &MetricsAgent{
		collector:      collector,
		reporter:       reporter,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		rateLimit:      rateLimit,
	}
}

func (a *MetricsAgent) worker(id int, jobs <-chan []models.Metrics) {
	log.Printf("Worker %d запущен", id)
	for metrics := range jobs {
		log.Printf("Worker %d: получил работу, отправляю %d метрик.", id, len(metrics))
		if err := a.reporter.Report(metrics); err != nil {
			log.Printf("Worker %d: не удалось отправить метрики: %v", id, err)
		}
	}
	log.Printf("Worker %d: канал jobs закрыт, завершаю работу.", id)
}

func (a *MetricsAgent) pollMetrics(ctx context.Context) {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	// Первоначальный сбор метрик
	a.collector.Poll()
	a.collector.PollGopsutil()

	for {
		select {
		case <-ticker.C:
			a.collector.Poll()
			a.collector.PollGopsutil()
		case <-ctx.Done():
			return
		}
	}
}

func (a *MetricsAgent) scheduleReports(ctx context.Context, jobs chan<- []models.Metrics) {
	ticker := time.NewTicker(a.reportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := a.collector.GetMetrics()
			if len(metrics) > 0 {
				jobs <- metrics
			}
		case <-ctx.Done():
			log.Println("Отправка финального отчета...")
			metrics := a.collector.GetMetrics()
			if len(metrics) > 0 {
				jobs <- metrics
			}
			return
		}
	}
}

func (a *MetricsAgent) Run(ctx context.Context) {
	var wg sync.WaitGroup
	jobs := make(chan []models.Metrics, a.rateLimit)

	for w := 1; w <= a.rateLimit; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			a.worker(workerID, jobs)
		}(w)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.pollMetrics(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(jobs)
		a.scheduleReports(ctx, jobs)
	}()

	log.Println("Агент запущен.")

	<-ctx.Done()
	log.Println("Контекст отменен. Ожидание завершения горутин...")

	wg.Wait()
	log.Println("Все горутины агента завершены.")
}

var (
	serverAddress  string
	reportInterval int
	pollInterval   int
	hashKey        string
	rateLimit      int
)

type config struct {
	serverAddress  string
	pollInterval   time.Duration
	reportInterval time.Duration
	hashKey        string
	rateLimit      int
}

const (
	defaultServerAddress  = "localhost:8080"
	defaultReportInterval = 10
	defaultPollInterval   = 2
	defaultHashKey        = ""
	defaultRateLimit      = 1
)

func registerFlags() {
	flag.StringVar(&serverAddress, "a", defaultServerAddress, "server adress")
	flag.IntVar(&reportInterval, "r", defaultReportInterval, "report interval")
	flag.IntVar(&pollInterval, "p", defaultPollInterval, "poll interval")
	flag.StringVar(&hashKey, "k", defaultHashKey, "key for hashing")
	flag.IntVar(&rateLimit, "l", defaultRateLimit, "rate limit for concurrent requests")
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
	helper.AssignFromViperIfSet(&rateLimit, "RATE_LIMIT", viper.GetInt, defaultRateLimit)
	
	if !govalidator.IsURL(serverAddress) {
		log.Fatalf("некорректный URL сервера: %s", serverAddress)
	}

	return config{
		serverAddress:  serverAddress,
		reportInterval: time.Duration(reportInterval) * time.Second,
		pollInterval:   time.Duration(pollInterval) * time.Second,
		hashKey:        hashKey,
		rateLimit:      rateLimit,
	}
}

func main() {
	cfg := parseConfig()
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Получен сигнал: %v. Завершение работы...", sig)
		cancel()
	}()

	// Создание компонентов
	collector := agent.NewMetricCollector()
	reporter := agent.NewHTTPReporter(cfg.serverAddress, cfg.hashKey)

	// Создание и запуск агента
	metricsAgent := NewMetricsAgent(
		collector,
		reporter,
		cfg.pollInterval,
		cfg.reportInterval,
		cfg.rateLimit,
	)
	metricsAgent.Run(ctx) // Блокирующий вызов

	log.Println("Агент остановлен.")
}
