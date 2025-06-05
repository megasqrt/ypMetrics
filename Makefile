echo:
	go version

run_s:
	go run cmd/server/main.go

run_a:
	go run cmd/agent/main.go

build:
	go build -o server cmd/server/main.go
	go build -o agent cmd/agent/main.go