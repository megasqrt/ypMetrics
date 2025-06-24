package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"os"
	"flag"
	"github.com/stretchr/testify/assert"
)

func TestNewMetricsAgent(t *testing.T) {
	agent := NewMetricsAgent("localhost:8080", 2*time.Second, 10*time.Second)
	if agent.serverAddress != "localhost:8080" {
		t.Errorf("Expected serverAddress 'localhost:8080', got '%s'", agent.serverAddress)
	}
	if agent.pollInterval != 2*time.Second {
		t.Errorf("Expected pollInterval 2s, got %v", agent.pollInterval)
	}
	if agent.reportInterval != 10*time.Second {
		t.Errorf("Expected reportInterval 10s, got %v", agent.reportInterval)
	}
	if len(agent.metrics) != 0 {
		t.Errorf("Expected empty metrics map, got %v", agent.metrics)
	}
}

func TestIncrementPollCount(t *testing.T) {
	agent := NewMetricsAgent("localhost:8080", 1*time.Second, 1*time.Second)
	
	agent.incrementPollCount()
	if agent.metrics["PollCount"] != int64(1) {
		t.Errorf("Expected PollCount 1, got %v", agent.metrics["PollCount"])
	}
	
	agent.incrementPollCount()
	if agent.metrics["PollCount"] != int64(2) {
		t.Errorf("Expected PollCount 2, got %v", agent.metrics["PollCount"])
	}
}

func TestFormatMetricURL(t *testing.T) {
	agent := NewMetricsAgent("localhost:8080", 1*time.Second, 1*time.Second)
	
	gaugeURL := agent.formatMetricURL("TestGauge", 3.14)
	expectedGaugeURL := "http://localhost:8080/update/gauge/TestGauge/3.140000"
	if gaugeURL != expectedGaugeURL {
		t.Errorf("Expected URL '%s', got '%s'", expectedGaugeURL, gaugeURL)
	}
	
	counterURL := agent.formatMetricURL("TestCounter", int64(42))
	expectedCounterURL := "http://localhost:8080/update/counter/TestCounter/42"
	if counterURL != expectedCounterURL {
		t.Errorf("Expected URL '%s', got '%s'", expectedCounterURL, counterURL)
	}
	
	invalidURL := agent.formatMetricURL("TestInvalid", "string")
	if invalidURL != "" {
		t.Errorf("Expected empty URL for unsupported type, got '%s'", invalidURL)
	}
}

func TestCollectRuntimeMetrics(t *testing.T) {
	agent := NewMetricsAgent("localhost:8080", 1*time.Second, 1*time.Second)
	agent.collectRuntimeMetrics()
	
	requiredMetrics := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects",
	}
	
	for _, metric := range requiredMetrics {
		if _, ok := agent.metrics[metric]; !ok {
			t.Errorf("Expected metric '%s' not found in collected metrics", metric)
		}
	}
}

func TestSendMetrics(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	
	agent := NewMetricsAgent(ts.URL[7:], 1*time.Second, 1*time.Second) // ts.URL[7:] чтобы убрать "http://"
	agent.metrics["TestGauge"] = 3.14
	agent.metrics["TestCounter"] = int64(42)
	
	agent.sendMetrics()
}

func TestAgentRun(t *testing.T) {
	agent := NewMetricsAgent("localhost:8080", 100*time.Millisecond, 100*time.Millisecond)
	agent.Run()
	
	time.Sleep(300 * time.Millisecond)
	
	if _, ok := agent.metrics["PollCount"]; !ok {
		t.Error("Expected PollCount to be incremented after agent run")
	}
	if _, ok := agent.metrics["RandomValue"]; !ok {
		t.Error("Expected RandomValue to be set after agent run")
	}
}


func TestFlagParsing(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name            string
		args           []string
		wantAddress    string
		wantReportInt  time.Duration
		wantPollInt    time.Duration
	}{
		{
			name:           "default values",
			args:           []string{"cmd"},
			wantAddress:    "localhost:8080",
			wantReportInt:  10 * time.Second,
			wantPollInt:    2 * time.Second,
		},
		{
			name:           "custom values",
			args:           []string{"cmd", "-a=127.0.0.1:9090", "-r=5s", "-p=1s"},
			wantAddress:    "127.0.0.1:9090",
			wantReportInt:  5 * time.Second,
			wantPollInt:    1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(tt.name, flag.ContinueOnError)
			
			os.Args = tt.args
			
			var (
				serverAddress string
				reportInterval time.Duration
				pollInterval   time.Duration
			)
			
			flag.StringVar(&serverAddress, "a", "localhost:8080", "server address")
			flag.DurationVar(&reportInterval, "r", 10*time.Second, "report interval")
			flag.DurationVar(&pollInterval, "p", 2*time.Second, "poll interval")
			
			flag.Parse()
			

			assert.Equal(t, tt.wantAddress, serverAddress, "parameter address not pass")
			assert.Equal(t, tt.wantReportInt, reportInterval,"parameter report not pass")
			assert.Equal(t, tt.wantPollInt, pollInterval,"parameter poll not pass")
		})
	}
}
