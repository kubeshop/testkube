test-all: 
	go test -v -cover ./...

run-api-serer: 
	go run cmd/api-server/main.go
