package middlewares

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"io"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ypMetrics/internal/helper"
)

func TestHashMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	const succesKey = "supersecret"
	//const badKey = "supersecret"
	const hashHeader = "HashSHA256"

	requestBody := `{"id":"TestGauge","type":"gauge"}`
	requestHash, _ := helper.CalculateHash([]byte(requestBody), succesKey)

	responseBody := "OK"
	responseHash, _ := helper.CalculateHash([]byte(responseBody), succesKey)

	//badResponseHash, _ := helper.CalculateHash([]byte(responseBody), badKey)


	tests := []struct {
		name               string
		key                string
		requestHeader      string
		requestBody        string
		expectedBody       string
		expectedStatusCode int
		expectResponseHash bool
	}{
		{
			name:               "valid hash",
			key:                succesKey,
			requestHeader:      requestHash,
			requestBody:        requestBody,
			expectedStatusCode: http.StatusOK,
			expectResponseHash: true,
		},
		{
			name:               "invalid hash",
			key:                succesKey,
			requestHeader:      "invalid_hash",
			requestBody:        requestBody,
			expectedStatusCode: http.StatusBadRequest,
			expectResponseHash: false,
		},
		{
			name:               "missing hash header",
			key:                succesKey,
			requestHeader:      "",
			requestBody:        requestBody,
			expectedStatusCode: http.StatusBadRequest,
			expectResponseHash: false,
		},
		{
			name:               "no key provided (pass-through)",
			key:                "",
			requestHeader:      "", // No hash needed
			requestBody:        requestBody,
			expectedStatusCode: http.StatusOK,
			expectResponseHash: false,
		},
				{
			name:               "bad key",
			key:                "",
			requestHeader:      "", 
			requestBody:        requestBody,
			expectedStatusCode: http.StatusOK,
			expectedBody:       "OK",
			expectResponseHash: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString(tt.requestBody))
			if tt.requestHeader != "" {
				req.Header.Set(hashHeader, tt.requestHeader)
			}

			rr := httptest.NewRecorder()
			hMd:=NewHashMiddleware(tt.key)
			middleware := hMd.HashMiddleware(nextHandler)
			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			if tt.expectResponseHash {
				assert.Equal(t, responseHash, rr.Header().Get(hashHeader))
			} else {
				assert.Empty(t, rr.Header().Get(hashHeader))
			}

			if tt.expectedStatusCode == http.StatusOK {
				body, err := io.ReadAll(rr.Body)
				require.NoError(t, err)
				assert.Equal(t, responseBody, string(body))
			}
		})
	}
}