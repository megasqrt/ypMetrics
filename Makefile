ADDRESS=localhost:8080
REPORT_INTERVAL=5
POLL_INTERVAL=1
DATABASE_DSN = host=127.0.0.1 user=metric password=metric dbname=metric sslmode=disable
	
export

echo:
	go version

run_s:
	@echo $$ADDRESS
	@echo $$REPORT_INTERVAL
	@echo $$POLL_INTERVAL
	@echo $$DATABASE_DSN
	go run cmd/server/main.go

run_a:
	go run cmd/agent/main.go
	
test_a:
	go run cmd/agent/

tests:
	go test ./...

build:
	go build -o server cmd/server/main.go
	go build -o agent cmd/agent/main.go