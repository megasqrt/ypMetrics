package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"ypMetrics/internal/helper"
	"ypMetrics/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBStorage_UpdateGauge(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &DBStorage{db: db}
	metricName := "test_gauge"
	metricValue := 123.45

	mock.ExpectExec("INSERT INTO gauges").WithArgs(metricName, metricValue).WillReturnResult(sqlmock.NewResult(1, 1))

	storage.UpdateGauge(metricName, metricValue)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBStorage_UpdateCounter(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &DBStorage{db: db}
	metricName := "test_counter"
	metricValue := int64(10)
	expectedValue := int64(20)

	rows := sqlmock.NewRows([]string{"value"}).AddRow(expectedValue)
	mock.ExpectQuery("INSERT INTO counters").WithArgs(metricName, metricValue).WillReturnRows(rows)

	newValue := storage.UpdateCounter(metricName, metricValue)

	assert.Equal(t, expectedValue, newValue)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBStorage_GetAllMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &DBStorage{db: db}

	gaugeRows := sqlmock.NewRows([]string{"id", "value"}).
		AddRow("g1", 1.1).
		AddRow("g2", 2.2)
	counterRows := sqlmock.NewRows([]string{"id", "value"}).
		AddRow("c1", int64(1)).
		AddRow("c2", int64(2))

	mock.ExpectQuery("SELECT id, value FROM gauges").WillReturnRows(gaugeRows)
	mock.ExpectQuery("SELECT id, value FROM counters").WillReturnRows(counterRows)

	allMetrics := storage.GetAllMetrics()

	expectedGauges := map[string]float64{"g1": 1.1, "g2": 2.2}
	expectedCounters := map[string]int64{"c1": 1, "c2": 2}

	assert.Equal(t, expectedGauges, allMetrics["gauges"])
	assert.Equal(t, expectedCounters, allMetrics["counters"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBStorage_GetMetricsByTypeAndName(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &DBStorage{db: db}

	t.Run("found gauge", func(t *testing.T) {

		rows := sqlmock.NewRows([]string{"value"}).AddRow(123.45)
		mock.ExpectQuery("SELECT value FROM gauges").WithArgs("test_gauge").WillReturnRows(rows)

		val, err := storage.GetMetricsByTypeAndName("test_gauge", models.Gauge)
		require.NoError(t, err)
		assert.Equal(t, "123.45", string(val))
	})

	t.Run("found counter", func(t *testing.T) {

		rows := sqlmock.NewRows([]string{"value"}).AddRow(int64(10))
		mock.ExpectQuery("SELECT value FROM counters").WithArgs("test_counter").WillReturnRows(rows)

		val, err := storage.GetMetricsByTypeAndName("test_counter", models.Counter)
		require.NoError(t, err)
		assert.Equal(t, "10", string(val))
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT value FROM gauges").WithArgs("not_found").WillReturnError(sql.ErrNoRows)
		_, err := storage.GetMetricsByTypeAndName("not_found", models.Gauge)
		assert.Error(t, err)
	})
}

func TestDBStorage_UpdateMetricsBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &DBStorage{db: db}

	metrics := []models.Metrics{
		{ID: "g1", MType: models.Gauge, Value: helper.Float64Ptr(1.1)},
		{ID: "c1", MType: models.Counter, Delta: helper.Int64Ptr(1)},
		{ID: "g2", MType: models.Gauge, Value: helper.Float64Ptr(2.2)},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO gauges")
	mock.ExpectPrepare("INSERT INTO counters")
	mock.ExpectExec("INSERT INTO gauges").WithArgs("g1", 1.1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO counters").WithArgs("c1", int64(1)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO gauges").WithArgs("g2", 2.2).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = storage.UpdateMetricsBatch(metrics)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBStorage_Ping(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &DBStorage{db: db}

	mock.ExpectPing()

	err = storage.Ping(context.Background())
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func Test_dbErrorIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"serialization failure", &pgconn.PgError{Code: "40001"}, true},
		{"deadlock detected", &pgconn.PgError{Code: "40P01"}, true},
		{"connection failure", &pgconn.PgError{Code: "08006"}, true},
		{"non-retryable error", errors.New("some other error"), false},
		{"nil error", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, dbErrorIsRetryable(tt.err))
		})
	}
}

// func TestDBStorage_GetJSONMetricsByTypeAndName(t *testing.T) {
// 	db, mock, err := sqlmock.New()
// 	if err != nil {
// 		t.Fatalf("an error %s was not expected when opening a stub database connection", err)
// 	}
// 	defer db.Close()

// 	s := &DBStorage{db: db}

// 	t.Run("Get gauge metric", func(t *testing.T) {
// 		metricName := "test_gauge"
// 		metricValue := 123.45

// rows := sqlmock.NewRows([]string{"value"}).AddRow(metricValue)
// 		mock.ExpectQuery(`SELECT value FROM gauges WHERE id = $1`).WithArgs(metricName).WillReturnRows(rows)

// 		jsonData, err := s.GetJSONMetricsByTypeAndName(metricName, "gauge")
// 		assert.NoError(t, err)
// 		expectedJSON := fmt.Sprintf(`{"id":"%s","type":"gauge","value":%f}`, metricName, metricValue)
// 		assert.JSONEq(t, expectedJSON, string(jsonData))
// 	})

// 	t.Run("Get counter metric", func(t *testing.T) {
// 		metricName := "test_counter"
// 		metricValue := int64(123)

// rows := sqlmock.NewRows([]string{"value"}).AddRow(metricValue)
// 		mock.ExpectQuery(`SELECT value FROM counters WHERE id = $1`).WithArgs(metricName).WillReturnRows(rows)

// 		jsonData, err := s.GetJSONMetricsByTypeAndName(metricName, "counter")
// 		assert.NoError(t, err)
// 		expectedJSON := fmt.Sprintf(`{"id":"%s","type":"counter","delta":%d}`, metricName, metricValue)
// 		assert.JSONEq(t, expectedJSON, string(jsonData))
// 	})

// 	t.Run("Metric not found", func(t *testing.T) {
// 		metricName := "non_existent"
// 		mock.ExpectQuery(`SELECT value FROM gauges WHERE id = $1`).WithArgs(metricName).WillReturnError(sql.ErrNoRows)

// 		_, err := s.GetJSONMetricsByTypeAndName(metricName, "gauge")
// 		assert.Error(t, err)
// 	})
// }

func TestDBStorage_RetryableErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}

	t.Run("UpdateGauge with retry", func(t *testing.T) {
		metricName := "retry_gauge"
		metricValue := 10.5
		retryableErr := &pgconn.PgError{Code: "40001"} // Serialization Failure

		mock.ExpectExec("INSERT INTO gauges").
			WithArgs(metricName, metricValue).
			WillReturnError(retryableErr)

		mock.ExpectExec("INSERT INTO gauges").
			WithArgs(metricName, metricValue).
			WillReturnResult(sqlmock.NewResult(1, 1))

		s.UpdateGauge(metricName, metricValue)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateCounter with retry", func(t *testing.T) {
		metricName := "retry_counter"
		metricValue := int64(5)
		expectedValue := int64(10)
		retryableErr := &pgconn.PgError{Code: "40P01"} // Deadlock Detected

		mock.ExpectQuery("INSERT INTO counters").
			WithArgs(metricName, metricValue).
			WillReturnError(retryableErr)

		rows := sqlmock.NewRows([]string{"value"}).AddRow(expectedValue)
		mock.ExpectQuery("INSERT INTO counters").
			WithArgs(metricName, metricValue).
			WillReturnRows(rows)

		newValue := s.UpdateCounter(metricName, metricValue)

		assert.Equal(t, expectedValue, newValue)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// func TestNewDBStorage_MigrationFailure(t *testing.T) {
// 	db, _, err := sqlmock.New()
// 	require.NoError(t, err)
// defer db.Close()

// 	_, err = NewDBStorage(nil)
// 	assert.Error(t, err, "NewDBStorage with nil db should return an error")
// }

func TestDBStorage_UpdateMetricsBatch_TransactionRollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}

	metrics := []models.Metrics{
		{ID: "g1", MType: "gauge", Value: helper.Float64Ptr(1.0)},
		{ID: "c1", MType: "counter", Delta: helper.Int64Ptr(1)},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO gauges")
	mock.ExpectPrepare("INSERT INTO counters")
	mock.ExpectExec("INSERT INTO gauges").WithArgs("g1", 1.0).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO counters").WithArgs("c1", int64(1)).WillReturnError(fmt.Errorf("some db error"))
	mock.ExpectRollback()

	err = s.UpdateMetricsBatch(metrics)
	require.Error(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func Test_populateMetrics_RowError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dest := make(map[string]float64)
	tableName := "gauges"

	rows := sqlmock.NewRows([]string{"id", "value"}).
		AddRow("g1", 1.1).
		AddRow("g2", "not-a-float") // This will cause a scan error

	mock.ExpectQuery(fmt.Sprintf("SELECT id, value FROM %s", tableName)).WillReturnRows(rows)

	populateMetrics(db, tableName, dest)

	assert.Contains(t, dest, "g1")
	assert.NotContains(t, dest, "g2")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func Test_getMetricFromDB_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Test non-ErrNoRows error
	mock.ExpectQuery("SELECT value FROM gauges").WithArgs("any").WillReturnError(fmt.Errorf("a different error"))
	_, err = getMetricFromDB[float64](db, "gauges", "any")
	assert.Error(t, err)
	assert.NotEqual(t, sql.ErrNoRows, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBStorage_UpdateCounter_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}
	metricName := "error_counter"
	metricValue := int64(10)
	dbError := errors.New("DB error")

	mock.ExpectQuery("INSERT INTO counters").WithArgs(metricName, metricValue).WillReturnError(dbError)

	result := s.UpdateCounter(metricName, metricValue)
	assert.Equal(t, metricValue, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBStorage_GetMetricsByTypeAndName_InvalidType(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}
	_, err = s.GetMetricsByTypeAndName("any_name", "invalid_type")
	assert.Error(t, err)
}

func TestDBStorage_GetJSONMetricsByTypeAndName_InvalidType(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}
	_, err = s.GetJSONMetricsByTypeAndName("any_name", "invalid_type")
	assert.Error(t, err)
}

func TestDBStorage_UpdateGauge_Retry(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}
	name := "test_gauge"
	value := 1.23
	pgErr := &pgconn.PgError{Code: "40001"} // Serialization Failure

	mock.ExpectExec("INSERT INTO gauges").WithArgs(name, value).WillReturnError(pgErr)
	mock.ExpectExec("INSERT INTO gauges").WithArgs(name, value).WillReturnResult(sqlmock.NewResult(1, 1))

	s.UpdateGauge(name, value)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBStorage_UpdateCounter_Retry(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}
	name := "test_counter"
	value := int64(5)
	pgErr := &pgconn.PgError{Code: "40P01"} // Deadlock

	mock.ExpectQuery("INSERT INTO counters").WithArgs(name, value).WillReturnError(pgErr)
	mock.ExpectQuery("INSERT INTO counters").WithArgs(name, value).WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(5))

	s.UpdateCounter(name, value)

	require.NoError(t, mock.ExpectationsWereMet())
}

// func TestDBStorage_Ping_Retry(t *testing.T) {
// 	db, mock, err := sqlmock.New()
// 	require.NoError(t, err)
// 	defer db.Close()

// 	s := &DBStorage{db: db}
// 	pgErr := &pgconn.PgError{Code: "08006"} // Connection Failure

// 	mock.ExpectPing().WillReturnError(pgErr)
// 	mock.ExpectPing().WillReturnError(pgErr)
// 	mock.ExpectPing().WillReturnError(pgErr)
// 	mock.ExpectPing().WillReturnError(pgErr)

// 	err = s.Ping(context.Background())
// 	require.Error(t, err) // Expecting an error after retries are exhausted

// 	require.NoError(t, mock.ExpectationsWereMet())
// }

func TestDBStorage_UpdateMetricsBatch_Retry(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}
	metrics := []models.Metrics{{ID: "g1", MType: "gauge", Value: helper.Float64Ptr(1.0)}}
	pgErr := &pgconn.PgError{Code: "40001"} // Serialization Failure

	// First attempt fails
	mock.ExpectBegin().WillReturnError(pgErr)

	// Second attempt succeeds
	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO gauges")
	mock.ExpectPrepare("INSERT INTO counters")
	mock.ExpectExec("INSERT INTO gauges").WithArgs("g1", 1.0).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = s.UpdateMetricsBatch(metrics)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func Test_getMetricFromDB_Retry(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pgErr := &pgconn.PgError{Code: "55P03"} // Lock Not Available

	mock.ExpectQuery("SELECT value FROM gauges").WithArgs("g1").WillReturnError(pgErr)
	mock.ExpectQuery("SELECT value FROM gauges").WithArgs("g1").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(1.23))

	_, err = getMetricFromDB[float64](db, "gauges", "g1")
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func Test_populateMetrics_Retry(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dest := make(map[string]float64)
	pgErr := &pgconn.PgError{Code: "08003"} // Connection Does Not Exist

	mock.ExpectQuery("SELECT id, value FROM gauges").WillReturnError(pgErr)
	mock.ExpectQuery("SELECT id, value FROM gauges").WillReturnRows(sqlmock.NewRows([]string{"id", "value"}).AddRow("g1", 1.23))

	populateMetrics(db, "gauges", dest)

	assert.Contains(t, dest, "g1")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBStorage_UpdateGauge_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}
	dbErr := errors.New("exec error")

	mock.ExpectExec("INSERT INTO gauges").WithArgs("g", 1.0).WillReturnError(dbErr)

	// The error is logged, not returned, so we just check expectations
	s.UpdateGauge("g", 1.0)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBStorage_UpdateCounter_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}

	rows := sqlmock.NewRows([]string{"value"}).AddRow("not-a-number")

	mock.ExpectQuery("INSERT INTO counters").WithArgs("c", int64(1)).WillReturnRows(rows)

	// Should return the original value
	result := s.UpdateCounter("c", int64(1))
	assert.Equal(t, int64(1), result)

	require.NoError(t, mock.ExpectationsWereMet())
}

func Test_populateMetrics_RowsErr(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	dest := make(map[string]float64)
	rowsErr := errors.New("rows iteration error")

	rows := sqlmock.NewRows([]string{"id", "value"}).AddRow("g1", 1.1).RowError(0, rowsErr)
	mock.ExpectQuery("SELECT id, value FROM gauges").WillReturnRows(rows)

	populateMetrics(db, "gauges", dest)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBStorage_GetJSONMetricsByTypeAndName_MarshalError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}

	// Use a value that causes json.Marshal to fail
	metricName := "test_gauge"
	metricValue := string(make([]byte, 1<<20)) // A large string to potentially cause issues

	rows := sqlmock.NewRows([]string{"value"}).AddRow(metricValue)
	mock.ExpectQuery(`SELECT value FROM gauges WHERE id = $1`).WithArgs(metricName).WillReturnRows(rows)

	_, err = s.GetJSONMetricsByTypeAndName(metricName, "gauge")
	assert.Error(t, err) // Expecting a scan error because we are trying to scan a string into a float64
}

func TestDBStorage_UpdateMetricsBatch_PrepareError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}
	metrics := []models.Metrics{{ID: "g1", MType: "gauge", Value: helper.Float64Ptr(1.0)}}
	prepErr := errors.New("prepare failed")

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO gauges").WillReturnError(prepErr)
	mock.ExpectRollback()

	err = s.UpdateMetricsBatch(metrics)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to prepare gauge statement")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBStorage_UpdateMetricsBatch_CommitError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &DBStorage{db: db}
	metrics := []models.Metrics{{ID: "g1", MType: "gauge", Value: helper.Float64Ptr(1.0)}}
	commitErr := errors.New("commit failed")

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO gauges")
	mock.ExpectPrepare("INSERT INTO counters")
	mock.ExpectExec("INSERT INTO gauges").WithArgs("g1", 1.0).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(commitErr)

	err = s.UpdateMetricsBatch(metrics)
	require.Error(t, err)
	assert.Equal(t, commitErr, err)

	require.NoError(t, mock.ExpectationsWereMet())
}
