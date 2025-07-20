package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"ypMetrics/internal/helper"
	"ypMetrics/models"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(db *sql.DB) (Storage, error) {
	storage := &DBStorage{db: db}
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}
	return storage, nil
}

func (s *DBStorage) initSchema() error {
	driver, err := postgres.WithInstance(s.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not create postgres driver: %w", err)
	}


	m, err := migrate.NewWithDatabaseInstance("file://internal/store/migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return  fmt.Errorf("could not run up migrations: %w", err)
	}

	return nil
}

func (s *DBStorage) UpdateGauge(name string, value float64) {
	query := `
    INSERT INTO gauges (id, value)
    VALUES ($1, $2)
    ON CONFLICT (id) DO UPDATE SET value = $2;
    `
	err := helper.Retryer(func() error {
		_, err := s.db.Exec(query, name, value)
		return err },
		dbErrorIsRetryable)
	if err != nil {
		log.Printf("Error updating gauge in DB: %v", err)
	}
}

func (s *DBStorage) UpdateCounter(name string, value int64) int64 {
	query := `
    INSERT INTO counters (id, value)
    VALUES ($1, $2)
    ON CONFLICT (id) DO UPDATE SET value = counters.value + $2
    RETURNING value;
    `
	var newDelta int64
	err := helper.Retryer(func() error {
		err := s.db.QueryRow(query, name, value).Scan(&newDelta)
		return err },
		dbErrorIsRetryable)
	if err != nil {
		log.Printf("Error updating counter in DB: %v", err)
		return value
	}
	return newDelta
}

func populateMetrics[T float64 | int64](db *sql.DB, tableName string, dest map[string]T) {
	query := fmt.Sprintf("SELECT id, value FROM %s", tableName)
	var rows *sql.Rows
	var err error
	retryErr := helper.Retryer(func() error {
		rows, err = db.Query(query)
		if err!=nil{
			return err
		}
		return nil },
		dbErrorIsRetryable)
	if retryErr != nil {
		log.Printf("Error getting %s from DB: %v", tableName, retryErr)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var value T
		if err := rows.Scan(&id, &value); err != nil {
			log.Printf("Error scanning %s row: %v", tableName, err)
			continue
		}
		dest[id] = value
	}
	if err := rows.Err(); err != nil {
		log.Printf("Error during %s rows iteration: %v", tableName, err)
	}
}

func (s *DBStorage) GetAllMetrics() map[string]interface{} {
	gauges := make(map[string]float64)
	counters := make(map[string]int64)

	populateMetrics(s.db, "gauges", gauges)
	populateMetrics(s.db, "counters", counters)

	return map[string]interface{}{
		"gauges":   gauges,
		"counters": counters,
	}
}

func getMetricFromDB[T float64 | int64](db *sql.DB, tableName, name string) (*T, error) {
	var value T
	query := fmt.Sprintf("SELECT value FROM %s WHERE id = $1", tableName)
	err := helper.Retryer(func() error {
		return db.QueryRow(query, name).Scan(&value)
	}, dbErrorIsRetryable)

	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (s *DBStorage) GetMetricsByTypeAndName(name, mtype string) ([]byte, error) {
	switch mtype {
	case models.Gauge:
		value, err := getMetricFromDB[float64](s.db, "gauges", name)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("metric '%s' of type 'gauge' not found", name)
			}
			return nil, err
		}
		return []byte(fmt.Sprintf("%g", *value)), nil
	case models.Counter:
		value, err := getMetricFromDB[int64](s.db, "counters", name)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("metric '%s' of type 'counter' not found", name)
			}
			return nil, err
		}
		return []byte(fmt.Sprintf("%d", *value)), nil
	default:
		return nil, fmt.Errorf("invalid metric type")
	}
}

func (s *DBStorage) GetJSONMetricsByTypeAndName(name, mtype string) ([]byte, error) {
	metric := models.Metrics{ID: name, MType: mtype}

	switch mtype {
	case models.Gauge:
		value, err := getMetricFromDB[float64](s.db, "gauges", name)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				log.Printf("Error getting gauge %s from DB: %v", name, err)
			}
			return nil, fmt.Errorf("metric not found")
		}
		metric.Value = value
		return json.Marshal(metric)
	case models.Counter:
		value, err := getMetricFromDB[int64](s.db, "counters", name)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				log.Printf("Error getting counter %s from DB: %v", name, err)
			}
			return nil, fmt.Errorf("metric not found")
		}
		metric.Delta = value
		return json.Marshal(metric)
	default:
		return nil, fmt.Errorf("metric not found")
	}
}

func (s *DBStorage) UpdateMetricsBatch(metrics []models.Metrics) error {
	retryableFunc := func() error {
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()

		gaugeStmt, err := tx.PrepareContext(context.Background(), `
			INSERT INTO gauges (id, value) VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value;
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare gauge statement: %w", err)
		}
		defer gaugeStmt.Close()

		counterStmt, err := tx.PrepareContext(context.Background(), `
			INSERT INTO counters (id, value) VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET value = counters.value + EXCLUDED.value;
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare counter statement: %w", err)
		}
		defer counterStmt.Close()

		for _, m := range metrics {
			switch m.MType {
			case models.Gauge:
				if m.Value != nil {
					if _, err := gaugeStmt.ExecContext(context.Background(), m.ID, *m.Value); err != nil {
						return fmt.Errorf("failed to execute gauge statement for %s: %w", m.ID, err)
					}
				}
			case models.Counter:
				if m.Delta != nil {
					if _, err := counterStmt.ExecContext(context.Background(), m.ID, *m.Delta); err != nil {
						return fmt.Errorf("failed to execute counter statement for %s: %w", m.ID, err)
					}
				}
			}
		}

		return tx.Commit()
	}
	return helper.Retryer(retryableFunc, dbErrorIsRetryable)

}

func (s *DBStorage) Ping(ctx context.Context) error {
	return helper.Retryer(func() error {
		return s.db.PingContext(ctx)
	}, dbErrorIsRetryable)
}

func dbErrorIsRetryable(err error) bool {
		var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Полный список кодов ошибок: https://www.postgresql.org/docs/current/errcodes-appendix.html
		switch pgErr.Code {
		// Ошибки, связанные с временной недоступностью или проблемами соединения.
		case pgerrcode.AdminShutdown,      // 57P01: Сервер закрывает соединение.
			pgerrcode.CannotConnectNow,     // 57P03: Сервер еще не готов принимать подключения.
			pgerrcode.ConnectionFailure,    // 08006: Обрыв соединения.
			pgerrcode.ConnectionDoesNotExist, // 08003: Соединение не существует.
			pgerrcode.TooManyConnections:   // 53300: Слишком много подключений.
			return true

		// Ошибки, связанные с конфликтами транзакций, которые можно разрешить повторной попыткой.
		case pgerrcode.SerializationFailure, // 40001: Ошибка сериализации транзакции.
			pgerrcode.DeadlockDetected,     // 40P01: Обнаружена взаимная блокировка (deadlock).
			pgerrcode.LockNotAvailable:       // 55P03: Ресурс заблокирован другой транзакцией.
			return true
		}
	}

	return false
}