package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"ypMetrics/models"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

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
	_, err := s.db.Exec(query, name, value)
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
	err := s.db.QueryRow(query, name, value).Scan(&newDelta)
	if err != nil {
		log.Printf("Error updating counter in DB: %v", err)
		return value
	}
	return newDelta
}

func (s *DBStorage) GetAllMetrics() map[string]interface{} {
	gauges := make(map[string]float64)
	counters := make(map[string]int64)

	// Запрос для gauges
	gaugeRows, err := s.db.Query("SELECT id, value FROM gauges")
	if err != nil {
		log.Printf("Error getting gauges from DB: %v", err)
	} else {
		defer gaugeRows.Close()
		for gaugeRows.Next() {
			var id string
			var value float64
			if err := gaugeRows.Scan(&id, &value); err != nil {
				log.Printf("Error scanning gauge row: %v", err)
				continue
			}
			gauges[id] = value
		}
		if err := gaugeRows.Err(); err != nil {
			log.Printf("Error during gauge rows iteration: %v", err)
		}
	}

	// Запрос для counters
	counterRows, err := s.db.Query("SELECT id, value FROM counters")
	if err != nil {
		log.Printf("Error getting counters from DB: %v", err)
	} else {
		defer counterRows.Close()
		for counterRows.Next() {
			var id string
			var value int64
			if err := counterRows.Scan(&id, &value); err != nil {
				log.Printf("Error scanning counter row: %v", err)
				continue
			}
			counters[id] = value
		}
		if err := counterRows.Err(); err != nil {
			log.Printf("Error during counter rows iteration: %v", err)
		}
	}

	return map[string]interface{}{
		"gauges":   gauges,
		"counters": counters,
	}
}

func (s *DBStorage) GetMetricsByTypeAndName(name, mtype string) ([]byte, error) {
	switch mtype {
	case models.Gauge:
		var value sql.NullFloat64
		err := s.db.QueryRow("SELECT value FROM gauges WHERE id = $1", name).Scan(&value)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("metric '%s' of type 'gauge' not found", name)
			}
			return nil, err
		}
		if value.Valid {
			return []byte(fmt.Sprintf("%g", value.Float64)), nil
		}
	case models.Counter:
		var delta sql.NullInt64
		err := s.db.QueryRow("SELECT value FROM counters WHERE id = $1", name).Scan(&delta)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("metric '%s' of type 'counter' not found", name)
			}
			return nil, err
		}
		if delta.Valid {
			return []byte(fmt.Sprintf("%d", delta.Int64)), nil
		}
	default:
		return nil, fmt.Errorf("invalid metric type")
	}
	return nil, fmt.Errorf("metric '%s' of type '%s' not found", name, mtype)
}

func (s *DBStorage) GetJSONMetricsByTypeAndName(name, mtype string) ([]byte, error) {
	var metric models.Metrics
	metric.ID = name
	metric.MType = mtype

	switch mtype {
	case models.Gauge:
		var value float64
		if err := s.db.QueryRow("SELECT value FROM gauges WHERE id = $1", name).Scan(&value); err == nil {
			metric.Value = &value
			return json.Marshal(metric)
		}
	case models.Counter:
		var delta int64
		if err := s.db.QueryRow("SELECT value FROM counters WHERE id = $1", name).Scan(&delta); err == nil {
			metric.Delta = &delta
			return json.Marshal(metric)
		}
	}
	return nil, fmt.Errorf("metric not found")
}

func (s *DBStorage) UpdateMetricsBatch(metrics []models.Metrics) error {
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
        ON CONFLICT (id) DO UPDATE SET value = counter.value + EXCLUDED.value;
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

func (s *DBStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
