NAME ?= testkube-executor-tracetest
BIN_DIR ?= $(HOME)/bin

help: Makefile ## show list of commands
	@echo "Choose a command run:"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-40s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

build: ## build executable locally
	go build -o $(BIN_DIR)/$(NAME) cmd/agent/main.go

.PHONY: test cover build

run: ## run the executor locally
	EXECUTOR_PORT=8082 go run cmd/agent/main.go

docker-build: ## build docker image
	docker build -t kubeshop/$(NAME) -f build/agent/Dockerfile .

test: ## run unit tests
	go test ./... -cover

cover: ## run coverage tests
	@go test -failfast -count=1 -v -tags test  -coverprofile=./testCoverage.txt ./... && go tool cover -html=./testCoverage.txt -o testCoverage.html && rm ./testCoverage.txt
	open testCoverage.html

version-bump: version-bump-patch

version-bump-patch:
	go run cmd/tools/main.go bump -k patch

version-bump-minor:
	go run cmd/tools/main.go bump -k minor

version-bump-major:
	go run cmd/tools/main.go bump -k major

version-bump-dev:
	go run cmd/tools/main.go bump --dev