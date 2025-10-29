package misc

import (
	"time"
)

type Config struct {
	ServerAddress     string
	GRPCServerAddress string
	StoreInterval     time.Duration
	FileStoragePath   string
	Restore           bool
	DatabaseDSN       string
	HashKey           string
	CryptoKey         string
	ConfigPath        string
	TrustedSubnet     string
}
