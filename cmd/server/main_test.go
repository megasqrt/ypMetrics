package main

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	testCases := []struct {
		name             string
		args             []string
		env              map[string]string
		expectedAddress  string
		expectedInterval time.Duration
		expectedPath     string
		expectedRestore  bool
		expectedDB       string
		expectedKey      string
	}{
		{
			name:             "default values",
			args:             []string{"cmd"},
			env:              nil,
			expectedAddress:  defaultServerAddress,
			expectedInterval: defaultStoreInterval * time.Second,
			expectedPath:     defaultFileStoragePath,
			expectedRestore:  defaultRestore,
			expectedDB:       defaultDatabaseDSN,
			expectedKey:      defaultHashKey,
		},
		{
			name:             "flag values",
			args:             []string{"cmd", "-a", "localhost:9090", "-i", "15", "-f", "/tmp/test.json", "-r=false", "-d", "test-dsn", "-k", "test-key"},
			env:              nil,
			expectedAddress:  "localhost:9090",
			expectedInterval: 15 * time.Second,
			expectedPath:     "/tmp/test.json",
			expectedRestore:  false,
			expectedDB:       "test-dsn",
			expectedKey:      "test-key",
		},
		{
			name: "env values",
			args: []string{"cmd"},
			env: map[string]string{
				"ADDRESS":           "localhost:7070",
				"STORE_INTERVAL":    "25",
				"FILE_STORAGE_PATH": "/tmp/env.json",
				"RESTORE":           "false",
				"DATABASE_DSN":      "env-dsn",
				"KEY":               "env-key",
			},
			expectedAddress:  "localhost:7070",
			expectedInterval: 25 * time.Second,
			expectedPath:     "/tmp/env.json",
			expectedRestore:  false,
			expectedDB:       "env-dsn",
			expectedKey:      "env-key",
		},
		{
			name: "flags_override_env",
			args: []string{"cmd", "-a", "localhost:9999", "-k", "flag-key"},
			env: map[string]string{
				"ADDRESS": "localhost:7777", // This will be ignored
				"KEY":     "env-key",        // This will be ignored
			},
			expectedAddress:  "localhost:9999", // Flag value should win
			expectedInterval: defaultStoreInterval * time.Second,
			expectedPath:     defaultFileStoragePath,
			expectedRestore:  true,
			expectedDB:       "",
			expectedKey:      "flag-key", // Flag value should win
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
			assert.Equal(t, tc.expectedAddress, cfg.ServerAddress)
			assert.Equal(t, tc.expectedInterval, cfg.StoreInterval)
			assert.Equal(t, tc.expectedPath, cfg.FileStoragePath)
			assert.Equal(t, tc.expectedRestore, cfg.Restore)
			assert.Equal(t, tc.expectedDB, cfg.DatabaseDSN)
			assert.Equal(t, tc.expectedKey, cfg.HashKey)
		})
	}
}
