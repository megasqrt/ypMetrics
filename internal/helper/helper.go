package helper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

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
			fmt.Printf("Error not retryable: %v. ", err)
			return err
		}

		if i < retryCount {
			fmt.Printf("Retrying after error: %v. Waiting %s", err, delay)
			time.Sleep(delay)
		}
	}
	return err
}

func CalculateHash(data []byte, key string) (string, error) {
	if key == "" {
		return "", nil
	}
	h := hmac.New(sha256.New, []byte(key))
	_, err := h.Write(data)
	if err != nil {
		return "", fmt.Errorf("failed to write to hmac: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}