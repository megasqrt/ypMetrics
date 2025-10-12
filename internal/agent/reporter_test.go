package agent

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"ypMetrics/internal/helper"
	"ypMetrics/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPReporter_Report(t *testing.T) {
	metricsToSend := []models.Metrics{
		{ID: "TestGauge", MType: "gauge", Value: helper.Ptr(3.14)},
		{ID: "TestCounter", MType: "counter", Delta: helper.Ptr(int64(42))},
	}

	var singleCalled, batchCalled, pingCalled int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch r.URL.Path {
		case "/ping":
			pingCalled++
			w.WriteHeader(http.StatusOK)
		case "/update/":
			singleCalled++
			// Для простоты теста просто считаем вызовы, не проверяя тело каждого запроса
			w.WriteHeader(http.StatusOK)
		case "/updates/":
			batchCalled++
			assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
			gz, err := gzip.NewReader(r.Body)
			require.NoError(t, err)
			defer gz.Close()

			var received []models.Metrics
			err = json.NewDecoder(gz).Decode(&received)
			require.NoError(t, err)
			assert.ElementsMatch(t, metricsToSend, received)
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
	}))
	defer server.Close()

	reporter := NewHTTPReporter(server.Listener.Addr().String(), "", "")
	t.Run("first report sends single", func(t *testing.T) {
		err := reporter.Report(metricsToSend)
		require.NoError(t, err)

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 1, pingCalled, "ping should be called once")
		assert.Equal(t, len(metricsToSend), singleCalled, "sendSingle should be called for each metric")
		assert.Equal(t, 0, batchCalled, "sendBatch should not be called")

		// Сбрасываем счетчики для следующего теста
		pingCalled = 0
		singleCalled = 0
	})
	t.Run("second report sends batch", func(t *testing.T) {
		err := reporter.Report(metricsToSend)
		require.NoError(t, err)

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 1, pingCalled, "ping should be called once")
		assert.Equal(t, 0, singleCalled, "sendSingle should not be called")
		assert.Equal(t, 1, batchCalled, "sendBatch should be called once")
	})
	t.Run("report fails if ping fails", func(t *testing.T) {
		// Создаем сервер, который всегда проваливает ping
		failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/ping" {
				http.Error(w, "ping failed", http.StatusInternalServerError)
			}
		}))
		defer failingServer.Close()

		failingReporter := NewHTTPReporter(failingServer.Listener.Addr().String(), "", "")
		err := failingReporter.Report(metricsToSend)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "report failed")
	})
}
