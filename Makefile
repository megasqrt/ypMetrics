echo:
	go version

run_s:
	go run cmd/server/main.go

run_a:
	go run cmd/agent/main.go
	
test_a:
	go run cmd/agent/

build:
	go build -o server cmd/server/main.go
	go build -o agent cmd/agent/main.go