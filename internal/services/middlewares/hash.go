package middlewares

import (
	"io"
	"net/http"
	"ypMetrics/internal/helper"

	"bytes"
	//"crypto/hmac"
	//"encoding/hex"

	"github.com/rs/zerolog/log"
)

type hashMiddleware struct {
	key string
}

type hashResponseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	key string
	statusCodeSet bool
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
		
		clientHashString := r.Header.Get("HashSHA256")
		if  clientHashString == ""{
			log.Print("missing hash header")
			next.ServeHTTP(w, r)
			return 
		}

		// clientHashByte, err := hex.DecodeString(clientHashString)
		// if err != nil {
		// 	http.Error(w, "Error decode request hash", http.StatusBadRequest)
		// 	return
		// }

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


		//iter 14 При несовпадении сервер должен отбрасывать полученные данные
		//  и возвращать http.StatusBadRequest.
		// serverHash:= helper.CalculateHashByte(bodyBytes, hm.key)
		// if !hmac.Equal(clientHashByte, serverHash) {
		// 	http.Error(w, "Invalid request hash", http.StatusBadRequest)
		// 	return
		// }

		hrw := &hashResponseWriter{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
			key:            hm.key,
			
		}

		next.ServeHTTP(hrw, r)


	})
}


func (hrw *hashResponseWriter) WriteHeader(statusCode int) {
	hrw.ResponseWriter.WriteHeader(statusCode)
	//hw.statusCodeSet = true
}

//iter 14 При наличии ключа на этапе формирования ответа
//  сервер должен вычислять хеш и передавать его в HTTP-заголовке ответа
//  с именем HashSHA256.
func (hrw *hashResponseWriter) Write(b []byte) (int, error) {
	hrw.body.Write(b)
	if hrw.key == "" && hrw.body.Len() > 0{
		responseHash := helper.CalculateHashString(hrw.body.Bytes(), hrw.key)
		hrw.Header().Set("HashSHA256", responseHash)	
	}
	return hrw.ResponseWriter.Write(b)
}