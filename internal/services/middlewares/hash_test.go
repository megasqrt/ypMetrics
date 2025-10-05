package middlewares

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"ypMetrics/internal/helper"

	"github.com/stretchr/testify/assert"
)

const testKeyForHash = "my-secret-testing-key"

func simpleTestHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func TestHashMiddleware_WithKey(t *testing.T) {
	mw := NewHashMiddleware(testKeyForHash)
	handler := mw.HashMiddleware(http.HandlerFunc(simpleTestHandler))

	t.Run("should pass request with valid hash and not add response hash", func(t *testing.T) {
		body := []byte(`{"id":"test_gauge","type":"gauge","value":123.45}`)
		hash := helper.CalculateHashString(body, testKeyForHash)

		req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
		req.Header.Set("HashSHA256", hash)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, body, rr.Body.Bytes(), "Response body should match request body")
		assert.Empty(t, rr.Header().Get("HashSHA256"), "Response hash should not be set when key is present")
	})

	t.Run("should not reject request with invalid hash", func(t *testing.T) {
		body := []byte(`{"id":"test_gauge","type":"gauge","value":123.45}`)

		req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
		req.Header.Set("HashSHA256", "this-is-a-bad-hash")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Should not reject request with invalid hash")
	})

	t.Run("should process request with missing hash and not add response hash", func(t *testing.T) {
		body := []byte(`{"id":"test_gauge","type":"gauge","value":123.45}`)

		req := httptest.NewRequest("POST", "/", bytes.NewReader(body))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Empty(t, rr.Header().Get("HashSHA256"), "Response hash should not be set when key is present")
	})
}

func TestHashMiddleware_WithoutKey(t *testing.T) {
	mw := NewHashMiddleware("") // No key
	handler := mw.HashMiddleware(http.HandlerFunc(simpleTestHandler))

	t.Run("should do nothing if key is not provided", func(t *testing.T) {
		body := []byte(`{"id":"test_gauge","type":"gauge","value":123.45}`)

		req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
		req.Header.Set("HashSHA256", "some-hash")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, body, rr.Body.Bytes())
		assert.Empty(t, rr.Header().Get("HashSHA256"), "Response hash should not be set when key is empty")
	})
}