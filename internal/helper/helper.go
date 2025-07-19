package helper

import (
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