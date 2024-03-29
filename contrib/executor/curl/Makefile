.PHONY: test cover

NAME ?= curl
BIN_DIR ?= $(HOME)/bin

run:
	cat test/example.json | go run cmd/agent/main.go

build:
	go build -o $(BIN_DIR)/$(NAME)-executor cmd/agent/main.go

docker-build:
	docker build -t kubeshop/$(NAME)-runner:latest -f build/agent/Dockerfile .

install-swagger-codegen-mac:
	brew install swagger-codegen

test:
	go test ./... -cover

.PHONY: integration-test
integration-test:
	INTEGRATION=y gotestsum --format pkgname -- -run _Integration -cover ./...

test-e2e:
	go test --tags=e2e -v ./test/e2e

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
