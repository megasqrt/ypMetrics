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

	t.Run("sends batch when ping is successful", func(t *testing.T) {
		var batchReceived bool
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/ping":
				w.WriteHeader(http.StatusOK)
			case "/updates/":
				mu.Lock()
				batchReceived = true
				mu.Unlock()

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

		reporter := NewHTTPReporter(server.Listener.Addr().String())
		err := reporter.Report(metricsToSend)
		require.NoError(t, err)

		mu.Lock()
		assert.True(t, batchReceived, "batch endpoint should have been called")
		mu.Unlock()
	})

	t.Run("sends single metrics when ping fails", func(t *testing.T) {
		var receivedMetrics []models.Metrics
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/ping":
				http.Error(w, "ping failed", http.StatusInternalServerError)
			case "/update/":
				gz, err := gzip.NewReader(r.Body)
				require.NoError(t, err)
				defer gz.Close()

				var m models.Metrics
				err = json.NewDecoder(gz).Decode(&m)
				require.NoError(t, err)

				mu.Lock()
				receivedMetrics = append(receivedMetrics, m)
				mu.Unlock()

				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		reporter := NewHTTPReporter(server.Listener.Addr().String())
		err := reporter.Report(metricsToSend)
		require.NoError(t, err)

		mu.Lock()
		assert.ElementsMatch(t, metricsToSend, receivedMetrics, "all metrics should have been sent individually")
		mu.Unlock()
	})
}
