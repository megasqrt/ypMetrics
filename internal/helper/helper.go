package helper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func AssignFromViperIfSet[T comparable](dst *T, key string, getter func(string) T, defaultValue T) {
	if *dst == defaultValue {
		if viper.IsSet(key) {
			*dst = getter(key)
		}
	}
}

func Ptr[T any](v T) *T {
	return &v
}

const (
	delay      = 5 * time.Second
	retryCount = 2
)

func Retryer(f func() error, isRetryable func(error) bool) error {
	var err error
	for i := 0; i < retryCount; i++ {
		err = f()
		if err == nil {
			return nil
		}

		if !isRetryable(err) {
			log.Printf("Error not retryable: %v. ", err)
			return err
		}

		if i < retryCount {
			log.Printf("Retrying after error: %v. Waiting %s", err, delay)
			time.Sleep(delay)
		}
	}
	return err
}

func CalculateHashString(data []byte, key string) string {
	
	hash := CalculateHashByte( data,key)
	return hex.EncodeToString(hash)
}

func CalculateHashByte(data []byte, key string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	return h.Sum(data)
}

func JSONErrorWithBody(w http.ResponseWriter, status int, requestBody []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(requestBody))
}

func JSONErrorWithMesage(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func Float64Ptr(v float64) *float64 { return &v }
func Int64Ptr(v int64) *int64    { return &v }