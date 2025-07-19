
ADDRESS=localhost:8080
REPORT_INTERVAL=5
POLL_INTERVAL=1
DATABASE_DSN = host=127.0.0.1 user=metric password=metric dbname=metric sslmode=disable	
#export

ITER = 13

echo:
	go version

run_s:
	go run cmd/server/main.go

run_a:
	go run cmd/agent/main.go
	
test_all: build
	@for i in $$(seq 1 $(ITER)); do \
		echo " === RUNNING TESTS FOR ITERATION $$i === "; \
		./metricstest -test.v -test.run="^TestIteration$$i$$" -agent-binary-path=./agent -binary-path=./server -server-port=8080 -database-dsn="$$DATABASE_DSN" -source-path=.; \
		done

test_iter: build
#	./metricstest -test.v -test.run="^TestIteration$(i)$$" -agent-binary-path=./agent -binary-path=./server -source-path=.;
	./metricstest -test.v -test.run="^TestIteration$(i)$$" -agent-binary-path=./agent -binary-path=./server -server-port=8080 -database-dsn="$$DATABASE_DSN" -source-path=.; \

tests_local:
	go vet -vettool=$(which statictest) ./...
	go test ./...

build:
	go build -o server cmd/server/main.go
	go build -o agent cmd/agent/main.go