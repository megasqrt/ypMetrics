package middlewares

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingMiddleware(t *testing.T) {
	var logBuf bytes.Buffer
	originalLogger := log.Logger
	log.Logger = zerolog.New(&logBuf)
	defer func() { log.Logger = originalLogger }() 
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("response body"))
	})

	loggingHandler := LoggingMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test-uri", nil)
	rr := httptest.NewRecorder()

	loggingHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	assert.Equal(t, "response body", rr.Body.String())

	var logEntry struct {
		Level    string  `json:"level"`
		Method   string  `json:"method"`
		URI      string  `json:"uri"`
		Status   int     `json:"status"`
		Size     int     `json:"size"`
		Duration float64 `json:"duration"`
		Message  string  `json:"message"`
	}

	err := json.Unmarshal(logBuf.Bytes(), &logEntry)
	require.NoError(t, err, "Failed to unmarshal log output")

	assert.Equal(t, "info", logEntry.Level)
	assert.Equal(t, "GET", logEntry.Method)
	assert.Equal(t, "/test-uri", logEntry.URI)
	assert.Equal(t, http.StatusAccepted, logEntry.Status)
	assert.Equal(t, len("response body"), logEntry.Size)
	assert.True(t, logEntry.Duration > 0, "Duration should be positive")
	assert.Equal(t, "request processed", logEntry.Message)
}
