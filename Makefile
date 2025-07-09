echo:
	go version

run_s:
	go run cmd/server/main.go

run_a:
	go run cmd/agent/main.go
	
test_a:
	go run cmd/agent/

tests:
	go test ./...

ADDRESS=localhost:35755
REPORT_INTERVAL=5
POLL_INTERVAL=1
export

test_s_bin: 
	@echo $$ADDRESS
	@echo $$REPORT_INTERVAL
	@echo $$POLL_INTERVAL
	./server 

build:
	go build -o server cmd/server/main.go
	go build -o agent cmd/agent/main.go