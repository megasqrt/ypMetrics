package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ypMetrics/models"
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
	assert.Empty(t, agent.metrics, "Expected empty metrics slice")
}

func TestIncrementPollCount(t *testing.T) {
	agent := NewMetricsAgent("localhost:8080", 1*time.Second, 1*time.Second)

	agent.incrementPollCount()

	var pollCountValue int64
	found := false
	for _, m := range agent.metrics {
		if m.ID == "PollCount" {
			pollCountValue = *m.Delta
			found = true
			break
		}
	}
	assert.True(t, found, "PollCount metric should be created")
	assert.Equal(t, int64(1), pollCountValue, "Expected PollCount 1 after first call")

	// Второй вызов должен инкрементировать
	agent.incrementPollCount()
	for _, m := range agent.metrics {
		if m.ID == "PollCount" {
			pollCountValue = *m.Delta
			break
		}
	}
	assert.Equal(t, int64(2), pollCountValue, "Expected PollCount 2 after second call")
}

func TestCollectRuntimeMetrics(t *testing.T) {
	agent := NewMetricsAgent("localhost:8080", 1*time.Second, 1*time.Second)
	agent.collectRuntimeMetrics()

	requiredMetrics := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects",
	}

	collectedMetrics := make(map[string]bool)
	for _, m := range agent.metrics {
		collectedMetrics[m.ID] = true
	}

	for _, metricName := range requiredMetrics {
		assert.True(t, collectedMetrics[metricName], "Expected metric '%s' not found in collected metrics", metricName)
	}
}

func TestSendMetrics(t *testing.T) {
	var receivedMetrics []models.Metrics
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/update/", r.URL.Path)
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))

		gz, err := gzip.NewReader(r.Body)
		require.NoError(t, err)
		defer gz.Close()

		var m models.Metrics
		err = json.NewDecoder(gz).Decode(&m)
		require.NoError(t, err)

		receivedMetrics = append(receivedMetrics, m)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := NewMetricsAgent(server.URL[7:], 1*time.Second, 1*time.Second) // ts.URL[7:] чтобы убрать "http://"

	agent.metrics = []models.Metrics{
		{ID: "TestGauge", MType: "gauge", Value: ptrFloat64(3.14)},
		{ID: "TestCounter", MType: "counter", Delta: ptrInt64(42)},
	}

	agent.sendMetrics()

	assert.Len(t, receivedMetrics, 2)
	assert.Equal(t, "TestGauge", receivedMetrics[0].ID)
	assert.Equal(t, "TestCounter", receivedMetrics[1].ID)
}

func TestAgentRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := NewMetricsAgent(server.URL[7:], 50*time.Millisecond, 100*time.Millisecond)
	agent.Run()

	time.Sleep(120 * time.Millisecond)

	metricExists := func(id string) bool {
		for _, m := range agent.metrics {
			if m.ID == id {
				return true
			}
		}
		return false
	}

	assert.True(t, metricExists("PollCount"), "Expected PollCount to be present after agent run")
	assert.True(t, metricExists("RandomValue"), "Expected RandomValue to be present after agent run")
}

func TestFlagParsing(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name          string
		args          []string
		wantAddress   string
		wantReportInt int
		wantPollInt   int
	}{
		{
			name:          "default values",
			args:          []string{"cmd"},
			wantAddress:   "localhost:8080",
			wantReportInt: 10,
			wantPollInt:   2,
		},
		{
			name:          "custom values",
			args:          []string{"cmd", "-a=127.0.0.1:9090", "-r=5", "-p=1"},
			wantAddress:   "127.0.0.1:9090",
			wantReportInt: 5,
			wantPollInt:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet(tt.name, flag.ContinueOnError)

			var (
				addr      string
				reportInt int
				pollInt   int
			)

			fs.StringVar(&addr, "a", "localhost:8080", "server address")
			fs.IntVar(&reportInt, "r", 10, "report interval")
			fs.IntVar(&pollInt, "p", 2, "poll interval")

			err := fs.Parse(tt.args[1:])
			assert.NoError(t, err)

			assert.Equal(t, tt.wantAddress, addr)
			assert.Equal(t, tt.wantReportInt, reportInt)
			assert.Equal(t, tt.wantPollInt, pollInt)
		})
	}
}
