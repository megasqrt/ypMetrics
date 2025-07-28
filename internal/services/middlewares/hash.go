package middlewares

import (
	"bytes"
	"io"
	"net/http"
	"ypMetrics/internal/helper"

	"github.com/rs/zerolog/log"
)

type hashMiddleware struct {
	key string
}

type hashResponseWriter struct {
	http.ResponseWriter
	body       []byte
}

func (hrw *hashResponseWriter) Write(b []byte) (int, error) {
	hrw.body=append(hrw.body, b...)
	return hrw.ResponseWriter.Write(b)
}

func NewHashMiddleware(key string) *hashMiddleware {
	return &hashMiddleware{key: key}
}

func (hm hashMiddleware) HashMiddlewareCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("key is %s", hm.key)
		if hm.key == "" {
			next.ServeHTTP(w, r)
			return
		}
		
		clientHash := r.Header.Get("HashSHA256")
		if  clientHash != ""{
			log.Print("missing hash header")
			next.ServeHTTP(w, r)
			return 
		}

		rBody := bytes.NewBuffer(nil)
		_, err := io.Copy(rBody, r.Body)
		if err != nil {
			log.Print("error copy request body")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.Body = io.NopCloser(rBody)

		serverHash, err := helper.CalculateHash(rBody.Bytes(), hm.key)
		if err != nil {
			log.Print("cannot calculate request hash")
			helper.JSONErrorWithBody(w, http.StatusBadRequest, rBody.Bytes())
			return
		}

		if clientHash != serverHash {
			log.Print("invalid hash")
			helper.JSONErrorWithBody(w, http.StatusBadRequest, rBody.Bytes())
			return
		}

		// hrw := &hashResponseWriter{
		// 	ResponseWriter: w,
		// 	body:           bytes.NewBuffer(nil),
		// 	statusCode:     http.StatusOK,
		// }

		// next.ServeHTTP(hrw, r)
		// responseBody := hrw.body.Bytes()
		// responseHash, err := helper.CalculateHash(responseBody, hm.key)
		// if err != nil {
		// 	helper.JSONErrorWithBody(w, http.StatusBadRequest, rBody.Bytes())
		// 	log.Error().Err(err).Msg("Failed to calculate response hash")
		// 	http.Error(w, "failed to calculate response hash", http.StatusInternalServerError)
		// 	return
		// }
		// w.Header().Set("HashSHA256", responseHash)
		// w.WriteHeader(hrw.statusCode)
		// w.Write(responseBody)

		next.ServeHTTP(w, r)

	})
}

func (hm hashMiddleware) HashMiddlewareResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("key is %s", hm.key)
		if hm.key == "" {
			next.ServeHTTP(w, r)
			return
		}

		hrw := &hashResponseWriter{
			ResponseWriter: w,
			body:           []byte{},
		}

		next.ServeHTTP(hrw, r)

		 responseBody := hrw.body
		 responseHash, err := helper.CalculateHash(responseBody, hm.key)
		if err != nil {
			log.Error().Err(err).Msg("Failed to calculate response hash")
			http.Error(w, "failed to calculate response hash", http.StatusInternalServerError)
			return
		}
		w.Header().Set("HashSHA256", responseHash)
	})
}