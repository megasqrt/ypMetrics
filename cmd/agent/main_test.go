package main

import (
	"context"
	"flag"
	"os"
	"sync"
	"testing"
	"time"

	"ypMetrics/internal/agent"
	"ypMetrics/models"

	"github.com/stretchr/testify/assert"
)

var _ agent.Collector = (*mockCollector)(nil)

type mockCollector struct {
	pollCalls         int
	pollGopsutilCalls int
	mu                sync.Mutex
}

func (m *mockCollector) Poll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pollCalls++
}

func (m *mockCollector) PollGopsutil() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pollGopsutilCalls++
}

func (m *mockCollector) GetMetrics() []models.Metrics {
	return []models.Metrics{{ID: "dummy", MType: "gauge"}}
}

var _ agent.Reporter = (*mockReporter)(nil)

type mockReporter struct {
	reportCalls int
	mu          sync.Mutex
}

func (m *mockReporter) Report(metrics []models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reportCalls++
	return nil
}

func TestNewMetricsAgent(t *testing.T) {
	var collector agent.Collector = agent.NewMetricCollector()
	reporter := agent.NewHTTPReporter("localhost:8080", "", "")
	pollInterval := 2 * time.Second
	reportInterval := 10 * time.Second
	rateLimit := 1

	metricsAgent := NewMetricsAgent(collector, reporter, pollInterval, reportInterval, rateLimit)

	assert.Equal(t, collector, metricsAgent.collector)
	assert.Equal(t, reporter, metricsAgent.reporter)
	assert.Equal(t, rateLimit, metricsAgent.rateLimit)
	assert.Equal(t, pollInterval, metricsAgent.pollInterval)
	assert.Equal(t, reportInterval, metricsAgent.reportInterval)
}

func TestAgentRun(t *testing.T) {
	collector := &mockCollector{}
	reporter := &mockReporter{}

	// Используем короткие интервалы для теста
	pollInterval := 20 * time.Millisecond
	reportInterval := 40 * time.Millisecond
	rateLimit := 2

	metricsAgent := NewMetricsAgent(collector, reporter, pollInterval, reportInterval, rateLimit)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Millisecond)
	defer cancel()

	metricsAgent.Run(ctx)

	collector.mu.Lock()
	assert.GreaterOrEqual(t, collector.pollCalls, 4, "poll should be called multiple times")
	assert.GreaterOrEqual(t, collector.pollGopsutilCalls, 4, "pollGopsutil should be called multiple times")
	collector.mu.Unlock()

	reporter.mu.Lock()

	assert.GreaterOrEqual(t, reporter.reportCalls, 2, "report should be called multiple times")
	reporter.mu.Unlock()
}

func TestParseConfig(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name              string
		args              []string
		env               map[string]string
		expectedAddr      string
		expectedReport    time.Duration
		expectedPoll      time.Duration
		expectedRateLimit int
	}{
		{
			name:              "default values",
			args:              []string{"cmd"},
			env:               nil,
			expectedAddr:      "localhost:8080",
			expectedReport:    10 * time.Second,
			expectedPoll:      2 * time.Second,
			expectedRateLimit: 1,
		},
		{
			name:              "custom flag values",
			args:              []string{"cmd", "-a=127.0.0.1:9090", "-r=5", "-p=1", "-l=3"},
			env:               nil,
			expectedAddr:      "127.0.0.1:9090",
			expectedReport:    5 * time.Second,
			expectedPoll:      1 * time.Second,
			expectedRateLimit: 3,
		},
		{
			name: "env values",
			args: []string{"cmd"},
			env: map[string]string{
				"ADDRESS":         "env.host:1234",
				"REPORT_INTERVAL": "15",
				"POLL_INTERVAL":   "3",
				"RATE_LIMIT":      "5",
			},
			expectedAddr:      "env.host:1234",
			expectedReport:    15 * time.Second,
			expectedPoll:      3 * time.Second,
			expectedRateLimit: 5,
		},
		{
			name: "flags override env values",
			args: []string{"cmd", "-a=flag.host:5678"},
			env: map[string]string{
				"ADDRESS": "env.host:1234",
			},
			expectedAddr:      "flag.host:5678",
			expectedReport:    10 * time.Second,
			expectedPoll:      2 * time.Second,
			expectedRateLimit: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			registerFlags()

			os.Args = tt.args
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg := parseConfig()

			assert.Equal(t, tt.expectedAddr, cfg.serverAddress)
			assert.Equal(t, tt.expectedReport, cfg.reportInterval)
			assert.Equal(t, tt.expectedPoll, cfg.pollInterval)
			assert.Equal(t, tt.expectedRateLimit, cfg.rateLimit)
		})
	}
}
