package services

import (
	"errors"
	"log"
	"net/http"
	"time"
	. "ypMetrics/models"
)

func RetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var lastErr error
		for i := 0; i < 3; i++ { 
			rw := &responseWriterWrapper{ResponseWriter: w} 
			next.ServeHTTP(rw, r)

			if rw.status < 500 && rw.status != 0 { 
				return
			}

			if lastErr != nil {
				log.Printf("Attempt %d failed: %v", i, lastErr)
			}

			
			if lastErr != nil && errors.As(lastErr, &RetriableError{}) {
				time.Sleep(time.Duration(1+2*i) * time.Second) // 1s, 3s, 5s
				continue
			} else if rw.status >= 500 {
				log.Printf("Attempt %d failed with status code: %d", i, rw.status)
				time.Sleep(time.Duration(1+2*i) * time.Second) // 1s, 3s, 5s
				continue
			} else {
				if lastErr != nil {
					http.Error(w, lastErr.Error(), http.StatusInternalServerError)
				}
				return
			}
		}
		log.Printf("All retry attempts failed: %v", lastErr)
		if lastErr != nil {
			http.Error(w, lastErr.Error(), http.StatusInternalServerError)
		}
	})
}

type responseWriterWrapper struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterWrapper) Write(data []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(data)
}

