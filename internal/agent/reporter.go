package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"
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
	mu            sync.Mutex
	nextBatch     bool
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
	err := r.pingServer()
	if err != nil {
		return fmt.Errorf("report failed. %v", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.nextBatch {
		err = r.sendBatch(metrics)
		if err != nil {
			return fmt.Errorf("ошибка отправки пакета метрик: %w", err)
		}
	} else {
		err = r.sendSingle(metrics)
		if err != nil {
			return fmt.Errorf("ошибка отправки single метрик: %w", err)
		}
	}
	r.nextBatch = !r.nextBatch
	return nil
}

func (r *HTTPReporter) sendBatch(metrics []models.Metrics) error {
	url := fmt.Sprintf("http://%s/updates/", r.serverAddress)
	return r.sendGzippedJSON(url, metrics)
}

func (r *HTTPReporter) sendSingle(metrics []models.Metrics) error {
	url := fmt.Sprintf("http://%s/update/", r.serverAddress)
	var firstErr error
	for _, m := range metrics {
		log.Println(m)
		if err := r.sendGzippedJSON(url, m); err != nil {
			log.Printf("Ошибка отправки метрики %s: %v", m.ID, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (r *HTTPReporter) pingServer() error {
	err := helper.Retryer(func() error {
		resp, err := r.client.Get(fmt.Sprintf("http://%s/ping", r.serverAddress))
		if err != nil {
			fmt.Println("pingServer error")
			return err
		}
		fmt.Println("pingServer ok")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("ping failed with status: %s", resp.Status)
		}
		fmt.Println("pingServer status ok")
		return nil
	}, httpErrorIsRetryable)
	return err
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

	compressedData := buf.Bytes()

	retryErr := func() error {
		//req, err := http.NewRequest(http.MethodPost, url, &buf)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(compressedData))
		if err != nil {
			return fmt.Errorf("ошибка создания запроса: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")

		if r.hashKey != "" {
			//hash, err := helper.CalculateHash(jsonData, r.hashKey)
			hash, err := helper.CalculateHash(compressedData, r.hashKey)
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
