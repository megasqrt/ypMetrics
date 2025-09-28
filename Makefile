
SERVER_PORT=8080
ADDRESS=localhost:${SERVER_PORT}
TEMP_FILE=$(random tempfile)
KEY=secretKey
REPORT_INTERVAL=5
POLL_INTERVAL=1
DATABASE_DSN=postgres://metric:metric@localhost:5432/metric?sslmode=disable
#export

ITER = 14

echo:
	go version

run_s:
	KEY=$(KEY) DATABASE_DSN=$(DATABASE_DSN) go run cmd/server/main.go

run_a:
	KEY=$(KEY) go run cmd/agent/main.go
	
test_all: build
	@for i in $$(seq 1 $(ITER)); do \
		echo " === RUNNING TESTS FOR ITERATION $$i === "; \
		./metricstest -test.v -test.run="^TestIteration$$i$$" -agent-binary-path=./agent -binary-path=./server -server-port=8080 -database-dsn="$$DATABASE_DSN" -source-path=.; \
		done

test_iter: build
#	./metricstest -test.v -test.run="^TestIteration$(i)$$" -agent-binary-path=./agent -binary-path=./server -source-path=.;
	./metricstest -test.v -test.run="^TestIteration$(i)$$" -agent-binary-path=./agent -binary-path=./server -server-port=8080 -database-dsn=$(DATABASE_DSN) -source-path=.; 

test_14: build
	./metricstest -test.v -test.run="^TestIteration13$$" -agent-binary-path=./agent -binary-path=./server -server-port=8080 -database-dsn=$(DATABASE_DSN) -source-path=.; 
	./metricstest -test.v -test.run="^TestIteration14$$" -agent-binary-path=./agent -binary-path=./server -database-dsn=$(DATABASE_DSN) -key=${KEY} -server-port=$(SERVER_PORT) -source-path=.


tests_local:
	go vet -vettool=$(which statictest) ./...
	go test ./...

build:
	go build -o server cmd/server/main.go
	go build -o agent cmd/agent/main.go