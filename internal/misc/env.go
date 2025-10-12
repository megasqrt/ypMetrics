package misc

import (
	"time"
)

type Config struct {
	ServerAddress   string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	HashKey         string
	CryptoKey       string
}
