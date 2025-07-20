package agent

import (
	"bytes"
	"compress/gzip"
	"errors"
	"net"
	"os"
	"io"
	"syscall"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"ypMetrics/internal/helper"
	"ypMetrics/models"
)

type Reporter interface {
	Report(metrics []models.Metrics) error
}

type HTTPReporter struct {
	serverAddress string
	client        *http.Client
	hashKey       string
}

func NewHTTPReporter(serverAddress string, hashKey string) *HTTPReporter {
	return &HTTPReporter{
		serverAddress: serverAddress,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		hashKey: hashKey,
	}
}

func (r *HTTPReporter) Report(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	if r.pingServer() {
		err := r.sendBatch(metrics)
		if err != nil {
			return fmt.Errorf("ошибка отправки пакета метрик: %w", err)
		}
		return nil
	}

	log.Println("Server ping failed. Falling back to single metric updates.")
	return r.sendSingle(metrics)
}

func (r *HTTPReporter) sendBatch(metrics []models.Metrics) error {
	url := fmt.Sprintf("http://%s/updates/", r.serverAddress)
	return r.sendGzippedJSON(url, metrics)
}

func (r *HTTPReporter) sendSingle(metrics []models.Metrics) error {
	url := fmt.Sprintf("http://%s/update/", r.serverAddress)
	var firstErr error
	for _, m := range metrics {
		if err := r.sendGzippedJSON(url, m); err != nil {
			log.Printf("Ошибка отправки метрики %s: %v", m.ID, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (r *HTTPReporter) pingServer() bool {
	resp, err := r.client.Get(fmt.Sprintf("http://%s/ping", r.serverAddress))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (r *HTTPReporter) sendGzippedJSON(url string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга данных: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(jsonData); err != nil {
		gz.Close()
		return fmt.Errorf("ошибка сжатия данных: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("ошибка закрытия gzip writer: %w", err)
	}
	retryErr := func() error {
		req, err := http.NewRequest(http.MethodPost, url, &buf)
		if err != nil {
			return fmt.Errorf("ошибка создания запроса: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")

		if r.hashKey != "" {
			hash, err := helper.CalculateHash(jsonData, r.hashKey)
			if err != nil {
				return fmt.Errorf("ошибка вычисления хеша: %w", err)
			}
			req.Header.Set("HashSHA256", hash)
		}
	
		resp, err := r.client.Do(req)	
		
		if err != nil {
			if resp != nil {
				resp.Body.Close()
			}
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("сервер вернул статус не-OK: %s", resp.Status)
		}
		return nil
	}
	return helper.Retryer(retryErr, httpErrorIsRetryable)
}

func httpErrorIsRetryable(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var syscallErr *os.SyscallError
		if errors.As(opErr.Err, &syscallErr) {
			switch syscallErr.Err {
			case syscall.ECONNREFUSED, syscall.ECONNRESET:
				return true
			}
		}
	}

	if errors.Is(err, io.EOF) {
		return true
	}
	return false
}