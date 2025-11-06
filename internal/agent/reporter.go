package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"ypMetrics/internal/helper"
	"ypMetrics/models"
	pb "ypMetrics/proto"
)

type Reporter interface {
	Report(metrics []models.Metrics) error
}

type GRPCReporter struct {
	client pb.MetricsClient
	conn   *grpc.ClientConn
}

func NewGRPCReporter(serverAddress string) (*GRPCReporter, error) {
	conn, err := grpc.NewClient(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к gRPC серверу: %w", err)
	}

	client := pb.NewMetricsClient(conn)

	return &GRPCReporter{
		client: client,
		conn:   conn,
	}, nil
}

func (r *GRPCReporter) Report(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	stream, err := r.client.Update(context.Background())
	if err != nil {
		return fmt.Errorf("не удалось создать gRPC stream: %w", err)
	}

	for _, m := range metrics {
		pbMetric := &pb.Metric{
			Id:   &m.ID,
			Type: &m.MType,
		}

		switch m.MType {
		case "gauge":
			if m.Value != nil {
				pbMetric.Value = m.Value
			}
		case "counter":
			if m.Delta != nil {
				pbMetric.Delta = m.Delta
			}
		}

		if err := stream.Send(pbMetric); err != nil {
			return fmt.Errorf("ошибка отправки метрики по gRPC stream: %w", err)
		}
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("ошибка при закрытии gRPC stream: %w", err)
	}

	return nil
}

type HTTPReporter struct {
	serverAddress string
	client        *http.Client
	hashKey       string
	mu            sync.Mutex
	nextBatch     bool
	publicKey     *rsa.PublicKey
	localIP       string
}

func NewHTTPReporter(serverAddress string, hashKey string, cryptoKeyPath string) (*HTTPReporter, error) {
	var publicKey *rsa.PublicKey
	if cryptoKeyPath != "" {
		keyBytes, err := os.ReadFile(cryptoKeyPath)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения публичного ключа: %w", err)
		}
		block, _ := pem.Decode(keyBytes)
		if block == nil {
			return nil, errors.New("не удалось декодировать публичный ключ")
		}
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("ошибка парсинга публичного ключа: %w", err)
		}
		var ok bool
		publicKey, ok = pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("неверный тип публичного ключа")
		}
	}

	var localIP string
	ip, err := getOutboundIP()
	if err != nil {
		log.Printf("Не удалось определить IP-адрес, заголовок X-Real-IP не будет установлен: %v", err)
	} else {
		localIP = ip.String()
	}

	return &HTTPReporter{
		serverAddress: serverAddress,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		hashKey:   hashKey,
		localIP:   localIP,
		publicKey: publicKey,
	}, nil
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

	if r.publicKey != nil {
		encryptedData, err := rsa.EncryptPKCS1v15(rand.Reader, r.publicKey, jsonData)
		if err != nil {
			return fmt.Errorf("ошибка шифрования данных: %w", err)
		}
		jsonData = encryptedData
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
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(buf.Bytes()))
		if err != nil {
			return fmt.Errorf("ошибка создания запроса: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")

		if r.localIP != "" {
			req.Header.Set("X-Real-IP", r.localIP)
		}

		if r.hashKey != "" {
			hash := helper.CalculateHashString(jsonData, r.hashKey)

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

func getOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}
