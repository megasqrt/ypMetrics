package main

import (
	"context"
	"database/sql"
	"flag"
	"os/signal"
	"syscall"
	"time"

	"net/http"
	_ "net/http/pprof"
	"ypMetrics/internal/helper"
	"ypMetrics/internal/misc"
	"ypMetrics/internal/services"
	"ypMetrics/internal/store"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

const (
	defaultServerAddress   = "localhost:8080"
	defaultStoreInterval   = 300
	defaultFileStoragePath = "/tmp/metrics-db.json"
	defaultRestore         = true
	defaultDatabaseDSN     = ""
	defaultHashKey         = ""
	defaultCryptoKey       = ""
	defaultConfigPath      = ""
)

func parseConfig(log zerolog.Logger) misc.Config {
	var cfg misc.Config
	var storeInterval int

	flag.StringVar(&cfg.ServerAddress, "a", defaultServerAddress, "server address")
	flag.IntVar(&storeInterval, "i", int(defaultStoreInterval), "store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", defaultFileStoragePath, "file storage path")
	flag.BoolVar(&cfg.Restore, "r", defaultRestore, "restore from file on start")
	flag.StringVar(&cfg.DatabaseDSN, "d", defaultDatabaseDSN, "database DSN")
	flag.StringVar(&cfg.HashKey, "k", defaultHashKey, "key for hashing")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", defaultCryptoKey, "path to private key file")
	flag.StringVar(&cfg.ConfigPath, "c", defaultConfigPath, "path to config file")
	flag.Parse()

	// Сначала читаем конфиг из файла, если он указан
	configPathFromEnv := viper.Get("CONFIG")
	if cfg.ConfigPath != "" {
		viper.SetConfigFile(cfg.ConfigPath)
	} else if configPathFromEnv != nil {
		viper.SetConfigFile(configPathFromEnv.(string))
	}

	if viper.ConfigFileUsed() != "" {
		if err := viper.ReadInConfig(); err != nil {
			// Логируем ошибку, но не падаем, т.к. конфиг не обязателен
			log.Error().Err(err).Msg("Error reading config file")
		}
	}

	viper.AutomaticEnv()

	helper.AssignFromViperIfSet(&cfg.ServerAddress, "ADDRESS", viper.GetString, defaultServerAddress)
	helper.AssignFromViperIfSet(&storeInterval, "STORE_INTERVAL", viper.GetInt, defaultStoreInterval)
	helper.AssignFromViperIfSet(&cfg.FileStoragePath, "FILE_STORAGE_PATH", viper.GetString, defaultFileStoragePath)
	helper.AssignFromViperIfSet(&cfg.Restore, "RESTORE", viper.GetBool, defaultRestore)
	helper.AssignFromViperIfSet(&cfg.DatabaseDSN, "DATABASE_DSN", viper.GetString, defaultDatabaseDSN)
	helper.AssignFromViperIfSet(&cfg.HashKey, "KEY", viper.GetString, defaultHashKey)
	helper.AssignFromViperIfSet(&cfg.CryptoKey, "CRYPTO_KEY", viper.GetString, defaultCryptoKey)
	helper.AssignFromViperIfSet(&cfg.ConfigPath, "CONFIG", viper.GetString, defaultConfigPath)

	cfg.StoreInterval = time.Duration(storeInterval) * time.Second

	return cfg
}

func main() {
	log := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	helper.BuildInfoPrint()

	cfg := parseConfig(log)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var storage store.Storage
	var db *sql.DB
	var err error

	if cfg.DatabaseDSN != "" {
		log.Info().Msg("Using database storage.")

		db, err = sql.Open("pgx", cfg.DatabaseDSN)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to connect to the database")
		}
		defer db.Close()

		if err = db.PingContext(ctx); err != nil && cfg.DatabaseDSN != "" {
			log.Fatal().Err(err).Msg("cannot connect to db")
		}

		storage, err = store.NewDBStorage(db)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create DB storage")
		}
		log.Info().Msg("Successfully create database storage.")
	} else {
		log.Info().Msg("Using in-memory storage.")
		memStorage, err := store.NewMemStorage(cfg.FileStoragePath, cfg.StoreInterval, cfg.Restore)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create memory storage")
		}
		storage = memStorage
		memStorage.StartPeriodicSave(ctx)
		defer memStorage.SaveOnExit()
	}

	server := services.NewMetricServer(cfg, storage, log)

	go func() {
		log.Info().Msg("Starting pprof server on :6060")
		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Fatal().Err(err).Msg("pprof server error")
		}
	}()

	go func() {
		log.Info().Str("address", cfg.ServerAddress).Msg("Starting server")
		if err := server.ListenAndServe(); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("Shutting down server...")
	server.Shutdown(context.Background())

}
