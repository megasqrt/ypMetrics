package helper

import "github.com/spf13/viper"


func AssignFromViperIfSet[T any](dst *T, key string, getter func(string) T) {
	if viper.IsSet(key) {
		*dst = getter(key)
	}
}