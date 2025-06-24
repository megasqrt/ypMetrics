package services

import (
	"net/http"
	"time"
	"github.com/rs/zerolog/log"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		logger := log.With().
			Str("method", r.Method).
			Str("uri", r.RequestURI).
			Logger()

		lw := &loggingResponseWriter{ResponseWriter: w}

		next.ServeHTTP(lw, r)

		duration := time.Since(start)
		logger.Info().
			Int("status", lw.status).
			Int("size", lw.size).
			Dur("duration", duration).
			Msg("request processed")
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}