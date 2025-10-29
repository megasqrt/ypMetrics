package main

import (
	"flag"
	"os"
	"testing"
	"time"

	"ypMetrics/internal/misc"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	testCases := []struct {
		name     string
		args     []string
		env      map[string]string
		expected misc.Config
	}{
		{
			name: "default values",
			args: []string{"cmd"},
			env:  nil,
			expected: misc.Config{
				ServerAddress:     defaultServerAddress,
				GRPCServerAddress: defaultGRPCServerAddress,
				StoreInterval:     defaultStoreInterval * time.Second,
				FileStoragePath:   defaultFileStoragePath,
				Restore:           defaultRestore,
				DatabaseDSN:       defaultDatabaseDSN,
				HashKey:           defaultHashKey,
				CryptoKey:         defaultCryptoKey,
				TrustedSubnet:     defaultTrustedSubnet,
				ConfigPath:        defaultConfigPath,
			},
		},
		{
			name: "flag values",
			args: []string{"cmd", "-a", "localhost:9090", "-ga", "localhost:9091", "-i", "15", "-f", "/tmp/test.json", "-r=false", "-d", "test-dsn", "-k", "test-key", "-crypto-key", "/path/to/key", "-t", "192.168.1.0/24", "-c", "/etc/config.json"},
			env:  nil,
			expected: misc.Config{
				ServerAddress:     "localhost:9090",
				GRPCServerAddress: "localhost:9091",
				StoreInterval:     15 * time.Second,
				FileStoragePath:   "/tmp/test.json",
				Restore:           false,
				DatabaseDSN:       "test-dsn",
				HashKey:           "test-key",
				CryptoKey:         "/path/to/key",
				TrustedSubnet:     "192.168.1.0/24",
				ConfigPath:        "/etc/config.json",
			},
		},
		{
			name: "env values",
			args: []string{"cmd"},
			env: map[string]string{
				"ADDRESS":           "localhost:7070",
				"GRPC_ADDRESS":      "localhost:7071",
				"STORE_INTERVAL":    "25",
				"FILE_STORAGE_PATH": "/tmp/env.json",
				"RESTORE":           "false",
				"DATABASE_DSN":      "env-dsn",
				"KEY":               "env-key",
				"CRYPTO_KEY":        "/env/key",
				"TRUSTED_SUBNET":    "10.0.0.0/8",
				"CONFIG":            "/env/config.json",
			},
			expected: misc.Config{
				ServerAddress:     "localhost:7070",
				GRPCServerAddress: "localhost:7071",
				StoreInterval:     25 * time.Second,
				FileStoragePath:   "/tmp/env.json",
				Restore:           false,
				DatabaseDSN:       "env-dsn",
				HashKey:           "env-key",
				CryptoKey:         "/env/key",
				TrustedSubnet:     "10.0.0.0/8",
				ConfigPath:        "/env/config.json",
			},
		},
		{
			name: "flags_override_env",
			args: []string{"cmd", "-a", "localhost:9999", "-k", "flag-key"},
			env: map[string]string{
				"ADDRESS": "localhost:7777", // This will be ignored
				"KEY":     "env-key",        // This will be ignored
			},
			expected: misc.Config{
				ServerAddress:     "localhost:9999", // Flag value should win
				GRPCServerAddress: defaultGRPCServerAddress,
				StoreInterval:     defaultStoreInterval * time.Second,
				FileStoragePath:   defaultFileStoragePath,
				Restore:           defaultRestore,
				DatabaseDSN:       defaultDatabaseDSN,
				HashKey:           "flag-key",
				CryptoKey:         defaultCryptoKey,
				TrustedSubnet:     defaultTrustedSubnet,
				ConfigPath:        defaultConfigPath,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// The magic line from the agent's test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Set up args and env
			os.Args = tc.args
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			log := zerolog.Nop()
			// Call the function to be tested
			cfg := parseConfig(log)

			// Assertions
			assert.Equal(t, tc.expected, cfg)
		})
	}
}
