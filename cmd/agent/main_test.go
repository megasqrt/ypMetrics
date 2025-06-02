package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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