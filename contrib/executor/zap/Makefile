NAME ?= testkube-zap-executor
BIN_DIR ?= $(HOME)/bin
NAMESPACE ?= "default"

build:
	go build -o $(BIN_DIR)/$(NAME) cmd/agent/main.go 

.PHONY: test cover build

run: 
	EXECUTOR_PORT=8082 go run cmd/agent/main.go

docker-build: 
	docker build -t kubeshop/$(NAME) -f build/agent/Dockerfile .

test: 
	go test ./... -cover
	
cover: 
	@go test -failfast -count=1 -v -tags test  -coverprofile=./testCoverage.txt ./... && go tool cover -html=./testCoverage.txt -o testCoverage.html && rm ./testCoverage.txt 
	open testCoverage.html
