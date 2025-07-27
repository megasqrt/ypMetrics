package services

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"time"
	"ypMetrics/internal/helper"

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

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to create gzip reader", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz := gzip.NewWriter(w)
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
	})
}

type hashResponseWriter struct {
	w          http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (hrw *hashResponseWriter) Header() http.Header {
	return hrw.w.Header()
}

func (hrw *hashResponseWriter) Write(data []byte) (int, error) {
	if hrw.statusCode == 0 {
		hrw.statusCode = http.StatusOK
	}
	return hrw.body.Write(data)
}

func (hrw *hashResponseWriter) WriteHeader(statusCode int) {
	hrw.statusCode = statusCode
}

func HashMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("key is %s", key)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			if r.Body != http.NoBody {
				clientHash := r.Header.Get("HashSHA256")
				if clientHash == "" {
					msg:="missing hash header"
					log.Print(msg)
					JSONError(w,http.StatusBadRequest,msg)
					return
				}

				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					msg:="cannot read body"
					log.Print(msg)
					JSONError(w,http.StatusBadRequest,msg)
					return
				}
				r.Body.Close()
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				serverHash, err := helper.CalculateHash(bodyBytes, key)
				if err != nil {
					msg:="cannot calculate hash"
					log.Print(msg)
					JSONError(w,http.StatusBadRequest,msg)
					return
				}

				if clientHash != serverHash {
					log.Print("invalid hash")
					http.Error(w, "invalid hash", http.StatusBadRequest)
					return
				}
			}

			hrw := &hashResponseWriter{w: w, body: bytes.NewBuffer(nil)}
			next.ServeHTTP(hrw, r)

			if hrw.body.Len() > 0 {
				if responseHash, err := helper.CalculateHash(hrw.body.Bytes(), key); err == nil && responseHash != "" {
					w.Header().Set("HashSHA256", responseHash)
				}
			}

			if hrw.statusCode != 0 {
				w.WriteHeader(hrw.statusCode)
			}
			w.Write(hrw.body.Bytes())
		})
	}
}