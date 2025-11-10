package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"ypMetrics/internal/mocks"
	"ypMetrics/internal/services/middlewares"

	"github.com/rs/zerolog"

	"github.com/gorilla/mux"
)

// setupMuxRouter создает и настраивает маршрутизатор с мок-хранилищем для тестов.
func setupMuxRouter() (*Handler, *mux.Router) {
	mockStorage := &mocks.MockStorage{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}
	service := NewMetricService(mockStorage, zerolog.New(io.Discard))
	ipChecker, err := middlewares.NewIPChecker("")
	if err != nil {
		panic(fmt.Sprintf("failed to create ip checker for tests: %v", err))
	}
	ipCheckMiddleware := middlewares.NewIPCheckMiddleware(ipChecker)
	handler := NewHandler(service, zerolog.New(io.Discard), ipCheckMiddleware)

	router := mux.NewRouter()
	router.HandleFunc("/update/{type}/{name}/{value}", handler.updateHandler).Methods(http.MethodPost)
	router.HandleFunc("/value/{type}/{name}", handler.getMetricHandler).Methods(http.MethodGet)
	router.HandleFunc("/", handler.metricsHTMLHandler).Methods(http.MethodGet)
	router.HandleFunc("/update/", handler.UpdateMetricJSON).Methods(http.MethodPost)
	router.HandleFunc("/value/", handler.GetMetricJSON).Methods(http.MethodPost)

	return &handler, router
}

func ExampleHandler_updateHandler() {
	_, router := setupMuxRouter()

	// Пример обновления метрики типа gauge
	reqGauge, _ := http.NewRequest(http.MethodPost, "/update/gauge/TestGauge/123.45", nil)
	rrGauge := httptest.NewRecorder()
	router.ServeHTTP(rrGauge, reqGauge)

	fmt.Println(rrGauge.Code)

	// Пример обновления метрики типа counter
	reqCounter, _ := http.NewRequest(http.MethodPost, "/update/counter/TestCounter/10", nil)
	rrCounter := httptest.NewRecorder()
	router.ServeHTTP(rrCounter, reqCounter)

	fmt.Println(rrCounter.Code)

	// Output:
	// 200
	// 200
}

func ExampleHandler_getMetricHandler() {
	_, router := setupMuxRouter()

	// Сначала обновим метрику, чтобы было что получать
	updateReq, _ := http.NewRequest(http.MethodPost, "/update/gauge/TestGauge/98.6", nil)
	router.ServeHTTP(httptest.NewRecorder(), updateReq)

	// Теперь получим значение этой метрики
	getReq, _ := http.NewRequest(http.MethodGet, "/value/gauge/TestGauge", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, getReq)

	fmt.Println(rr.Code)
	fmt.Println(rr.Body.String())

	// Output:
	// 200
	// 98.6
}

func ExampleHandler_GetMetricJSON() {
	_, router := setupMuxRouter()

	// Сначала обновим метрику
	updateBody := `{"id":"TestCounterJSON","type":"counter","delta":50}`
	updateReq, _ := http.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), updateReq)

	// Теперь получим ее значение через JSON
	getBody := `{"id":"TestCounterJSON","type":"counter"}`
	getReq, _ := http.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString(getBody))
	getReq.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, getReq)

	fmt.Println(rr.Code)
	fmt.Println(rr.Body.String())

	// Output:
	// 200
	// {"id":"TestCounterJSON","type":"counter","delta":50}
}

func ExampleHandler_UpdateMetricJSON() {
	_, router := setupMuxRouter()

	// Пример обновления метрики через JSON
	jsonBody := `{"id":"TestGaugeJSON","type":"gauge","value":101.1}`
	req, _ := http.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	fmt.Println(rr.Code)
	fmt.Println(rr.Body.String())

	// Output:
	// 200
	// {"id":"TestGaugeJSON","type":"gauge","value":101.1}
}

func ExampleHandler_metricsHTMLHandler() {
	_, router := setupMuxRouter()

	// Добавим несколько метрик для отображения
	req1, _ := http.NewRequest(http.MethodPost, "/update/gauge/Temp/25.5", nil)
	router.ServeHTTP(httptest.NewRecorder(), req1)
	req2, _ := http.NewRequest(http.MethodPost, "/update/counter/Clicks/15", nil)
	router.ServeHTTP(httptest.NewRecorder(), req2)

	// Запросим главную страницу
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Для краткости выведем только содержательную часть ответа
	body := rr.Body.String()
	fmt.Println(strings.Contains(body, "<h2>Gauge Metrics</h2>"))
	fmt.Println(strings.Contains(body, "Temp:"))
	fmt.Println(strings.Contains(body, "25.50"))
	fmt.Println(strings.Contains(body, "<h2>Counter Metrics</h2>"))
	fmt.Println(strings.Contains(body, "Clicks:"))
	fmt.Println(strings.Contains(body, "15"))

	// Output:
	// true
	// true
	// true
	// true
	// true
	// true
}
