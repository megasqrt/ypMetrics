package middlewares

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Write(body)
	})

	gzipHandler := GzipMiddleware(testHandler)

	t.Run("sends_gzipped_and_receives_gzipped", func(t *testing.T) {
		var requestBody bytes.Buffer
		gzw := gzip.NewWriter(&requestBody)
		_, err := gzw.Write([]byte("test body"))
		require.NoError(t, err)
		gzw.Close()

		req := httptest.NewRequest("POST", "/", &requestBody)
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")

		rr := httptest.NewRecorder()
		gzipHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

		gz, err := gzip.NewReader(rr.Body)
		require.NoError(t, err)
		defer gz.Close()

		uncompressedBody, err := io.ReadAll(gz)
		require.NoError(t, err)
		assert.Equal(t, "test body", string(uncompressedBody))
	})

	t.Run("sends_ungzipped_and_receives_ungzipped", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", bytes.NewBufferString("plain body"))

		rr := httptest.NewRecorder()
		gzipHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Empty(t, rr.Header().Get("Content-Encoding")) // No gzip encoding
		assert.Equal(t, "plain body", rr.Body.String())
	})

	t.Run("sends_ungzipped_and_receives_gzipped", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", bytes.NewBufferString("plain body, gzipped response"))
		req.Header.Set("Accept-Encoding", "gzip")

		rr := httptest.NewRecorder()
		gzipHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

		gz, err := gzip.NewReader(rr.Body)
		require.NoError(t, err)
		defer gz.Close()

		uncompressedBody, err := io.ReadAll(gz)
		require.NoError(t, err)
		assert.Equal(t, "plain body, gzipped response", string(uncompressedBody))
	})
}
