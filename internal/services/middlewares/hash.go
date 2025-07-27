package middlewares

import (
	"bytes"
	"io"
	"net/http"
	"ypMetrics/internal/helper"

	"github.com/rs/zerolog/log"
)

type hashMiddleware struct{
	key string
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

func NewHashMiddleware(key string) *hashMiddleware{
	return &hashMiddleware{key: key}
}

func (hm hashMiddleware) HashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("key is %s", hm.key)
		if hm.key == "" {
			next.ServeHTTP(w, r)
			return
		}

		clientHash := r.Header.Get("HashSHA256")
		if clientHash == "" {
			log.Print("missing hash header")
			next.ServeHTTP(w, r)
			return
		}

		rBody:=bytes.NewBuffer(nil)
		_,err:=io.Copy(rBody,r.Body)
		if err !=nil{
			log.Print("error copy request body")
			w.WriteHeader(http.StatusBadRequest)
			return 
		}


		serverHash, err := helper.CalculateHash(rBody.Bytes(), hm.key)
		log.Print(serverHash) //TODO REMOVE
		if err != nil {
			log.Print("cannot calculate hash")
			helper.JSONErrorWithBody(w, http.StatusBadRequest, rBody.Bytes())
			return
		}

		if clientHash != serverHash {
			log.Print("invalid hash")
			helper.JSONErrorWithBody(w, http.StatusBadRequest, rBody.Bytes())
			return
		}

			// hrw := &hashResponseWriter{w: w, body: bytes.NewBuffer(nil)}
			// next.ServeHTTP(hrw, r)
			// if hrw.statusCode >= 400 {
			// 	JSONErrorWithBody(w, hrw.statusCode, bodyBytes)
			// 	return 
			// }


			// if hrw.body.Len() > 0 {
			// 	if responseHash, err := helper.CalculateHash(hrw.body.Bytes(), key); err == nil && responseHash != "" {
			// 		w.Header().Set("HashSHA256", responseHash)
			// 	}
			// }

			// if hrw.statusCode != 0 {
			// 	w.WriteHeader(hrw.statusCode)
			// }

			next.ServeHTTP(w,r)
		})
}
