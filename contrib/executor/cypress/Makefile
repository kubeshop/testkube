.PHONY: test cover build

CHART_NAME=cypress-executor
NAME ?= cypress
BIN_DIR ?= $(HOME)/bin
GITHUB_TOKEN ?= "SET_ME"
USER ?= $(USER)
NAMESPACE ?= "example-ns"
DATE ?= $(shell date -u --iso-8601=seconds)
COMMIT ?= $(shell git log -1 --pretty=format:"%h")

# TODO bump this port up - to be able to run multiple executors on devs machine
run-executor:
	EXECUTOR_PORT=8084 go run cmd/agent/main.go

run-mongo-dev:
	docker run -p 27017:27017 mongo


build:
	go build -o $(BIN_DIR)/$(NAME)-executor cmd/agent/main.go

docker-build-executor:
	docker build -t $(NAME)-executor -f build/agent/Dockerfile .

docker-build-runner:
	docker build -t kubeshop/$(NAME)-runner -f build/agent/Dockerfile .

install-swagger-codegen-mac:
	brew install swagger-codegen

test:
	go test ./... -cover

.PHONY: integration-test
integration-test:
	INTEGRATION=y gotestsum --format pkgname -- -run _Integration -cover ./...

cover:
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

prerelease:
	go run cmd/tools/main.go release -d -a $(CHART_NAME)

release:
	go run cmd/tools/main.go release -a $(CHART_NAME)

update-modules:
	go mod tidy
	go get -u ./...
