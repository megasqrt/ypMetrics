
SERVER_PORT=8080
ADDRESS=localhost:${SERVER_PORT}
TEMP_FILE=$(random tempfile)
KEY=secretKey
REPORT_INTERVAL=5
POLL_INTERVAL=1
DATABASE_DSN=postgres://metric:metric@localhost:5432/metric?sslmode=disable
#export

echo:
	go version

run_s:
	KEY=$(KEY) DATABASE_DSN=$(DATABASE_DSN) go run cmd/server/main.go

run_a:
	KEY=$(KEY) go run cmd/agent/main.go

tests:
	go vet -vettool=$(which statictest) ./...
	go test ./...
	go test -v -race ./...

build:
	go build -o server cmd/server/main.go
	go build -o agent cmd/agent/main.go

cover:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out 
#go tool cover -html=coverage.out -o coverage.html

mem_test:
	run_s &
	run_a &
	sleep 60 
	go tool pprof -http=":9090" -seconds=30 http://localhost:6060/debug/pprof/heap
	curl -o memprofile.pprof http://localhost:6060/debug/pprof/heap

pprof_compare:
	go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof 

fmtall:
	goimports -w ./.
