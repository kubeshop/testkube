REPOSITORY ?= kubeshop
NAME ?= testkube-jmeterd-executor
SLAVES_NAME ?= testkube-jmeterd-slave
LOCAL_TAG ?= 999.0.0
BIN_DIR ?= $(HOME)/bin

.PHONY: build
build:
	go build -o $(BIN_DIR)/$(NAME) cmd/agent/main.go

.PHONY: build-local-linux
build-local-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/runner cmd/agent/main.go

.PHONY: run
run:
	EXECUTOR_PORT=8082 go run cmd/agent/main.go ${run_args}

.PHONY: docker-build
docker-build:
	docker build -t $(REPOSITORY)/$(NAME) -f build/agent/Dockerfile .

.PHONY: docker-build-local
docker-build-local: build-local-linux
	docker build -t $(REPOSITORY)/$(NAME):$(LOCAL_TAG) -f build/agent/local.Dockerfile .

docker-build-slaves:
	docker build -t $(REPOSITORY)/$(SLAVES_NAME):$(LOCAL_TAG) -f build/slaves/Dockerfile .

.PHONY: kind-load-local
kind-load-local: build-local-linux
	kind load docker-image kubeshop/testkube-jmeterd-executor:999.0.0

.PHONY: test
test:
	go test ./... -cover

.PHONY: integration-test
integration-test:
	INTEGRATION=y gotestsum --format pkgname -- -run _Integration -cover ./...

.PHONY: cover
cover:
	@go test -failfast -count=1 -v -tags test  -coverprofile=./testCoverage.txt ./... && go tool cover -html=./testCoverage.txt -o testCoverage.html && rm ./testCoverage.txt
	open testCoverage.html

.PHONY: version-bump
version-bump: version-bump-patch

.PHONY: version-bump-patch
version-bump-patch:
	go run cmd/tools/main.go bump -k patch

.PHONY: version-bump-minor
version-bump-minor:
	go run cmd/tools/main.go bump -k minor

.PHONY: version-bump-major
version-bump-major:
	go run cmd/tools/main.go bump -k major

.PHONY: version-bump-dev
version-bump-dev:
	go run cmd/tools/main.go bump --dev
