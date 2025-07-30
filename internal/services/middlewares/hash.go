package middlewares

import (
	//"encoding/json"
	"io"
	"net/http"
	"ypMetrics/internal/helper"
	//"ypMetrics/models"
	"bytes"


	"github.com/rs/zerolog/log"
)

type hashMiddleware struct {
	key string
}

func NewHashMiddleware(key string) *hashMiddleware {
	return &hashMiddleware{key: key}
}


//iter 14 При наличии ключа во время обработки запроса сервер должен
//  проверять соответствие полученного и вычисленного хеша.	
func (hm hashMiddleware) HashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("key is %s", hm.key)
		if hm.key == "" {
			next.ServeHTTP(w, r)
			return
		}
		
		clientHash := r.Header.Get("HashSHA256")
		if  clientHash == ""{
			log.Print("missing hash header")
			next.ServeHTTP(w, r)
			return 
		}

		if r.Body == nil {
			next.ServeHTTP(w, r)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Print("error reading request body")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// var v []models.Metrics
		// err = json.Unmarshal(bodyBytes,&v)
		// if err != nil {
		// 	log.Print("cannot unmarshal request body")
		// 	helper.JSONErrorWithBody(w, http.StatusBadRequest, bodyBytes)
		// }

		serverHash, err := helper.CalculateHash(bodyBytes, hm.key)
		if err != nil {
			log.Print("cannot calculate request hash")
			helper.JSONErrorWithBody(w, http.StatusBadRequest, bodyBytes)
			return
		}

		//iter 14 При несовпадении сервер должен отбрасывать полученные данные
		//  и возвращать http.StatusBadRequest.
		if clientHash != serverHash {
			log.Print("invalid hash")
			helper.JSONErrorWithBody(w, http.StatusBadRequest, bodyBytes)
			return
		}

		hrw := &hashResponseWriter{
			ResponseWriter: w,
			body:           []byte{},
			key:            hm.key,
		}

		next.ServeHTTP(hrw, r)

	})
}


type hashResponseWriter struct {
	http.ResponseWriter
	body       []byte
	key string
}

func (hw *hashResponseWriter) WriteHeader(statusCode int) {

	hw.ResponseWriter.WriteHeader(statusCode)
}

//iter 14 При наличии ключа на этапе формирования ответа
//  сервер должен вычислять хеш и передавать его в HTTP-заголовке ответа
//  с именем HashSHA256.
func (hrw *hashResponseWriter) Write(b []byte) (int, error) {
	
	if hrw.key == "" {
		responseBody := hrw.body
		if len(responseBody) > 0 {
		responseHash, err := helper.CalculateHash(responseBody, hrw.key)
			
		if err != nil {
			log.Error().Err(err).Msg("Failed to calculate response hash")
			http.Error(hrw, "failed to calculate response hash", http.StatusInternalServerError)
		}
		hrw.Header().Set("HashSHA256", responseHash)	
		}
	}
	return hrw.ResponseWriter.Write(b)
}