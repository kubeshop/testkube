# Testkube Makefile
#
# This Makefile provides a comprehensive build system for the Testkube project,
# supporting cross-platform development, testing, and deployment workflows.
#
# Usage: make help

# ==================== Configuration ====================
# Disable built-in rules and variables for performance and clarity
MAKEFLAGS += --no-builtin-rules --no-builtin-variables

# Disable implicit rules and pattern rules to speed up Make
.SUFFIXES:
MAKEFLAGS += --no-print-directory

# Tell Make to not search in subdirectories for prerequisites
VPATH =

# Prevent Make from doing parallel execution and implicit rule searches
.NOTPARALLEL:

# Default shell configuration for consistent behavior across platforms
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

# Enable secondary expansion for advanced pattern rules
.SECONDEXPANSION:

# Delete targets on error to maintain clean state
.DELETE_ON_ERROR:

# Export all variables to sub-makes by default
.EXPORT_ALL_VARIABLES:

# Include .env file if it exists (won't fail if missing)
-include .env

# ==================== OS Detection ====================
# Detect operating system for platform-specific configurations
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Linux)
    OS := linux
    SED_INPLACE := sed -i
    OPEN_CMD := xdg-open
else ifeq ($(UNAME_S),Darwin)
    OS := darwin
    SED_INPLACE := sed -i ''
    OPEN_CMD := open
else ifeq ($(UNAME_S),Windows_NT)
    OS := windows
    SED_INPLACE := sed -i
    OPEN_CMD := start
else
    $(error Unsupported operating system: $(UNAME_S))
endif

# Architecture detection
ifeq ($(UNAME_M),x86_64)
    ARCH := amd64
else ifeq ($(UNAME_M),aarch64)
    ARCH := arm64
else ifeq ($(UNAME_M),arm64)
    ARCH := arm64
else
    ARCH := $(UNAME_M)
endif

# ==================== Project Variables ====================
# Core project configuration
PROJECT_NAME := testkube
CHART_NAME := api-server
NAMESPACE ?= testkube

# Version and build metadata
VERSION ?= 999.0.0-$(shell git log -1 --pretty=format:"%h" 2>/dev/null || echo "unknown")
COMMIT = $(shell git log -1 --pretty=format:"%h")
DATE = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
USER = $(shell whoami)

# Directory configuration
BUILD_DIR := build
DIST_DIR := dist
TMP_DIR := tmp
CONFIG_DIR := config
DOCS_DIR := docs

# Local binary directories
LOCALBIN ?= $(PWD)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# LOCALBIN_APP refers to the directory where application binaries are installed
LOCALBIN_APP ?= $(LOCALBIN)/app
$(LOCALBIN_APP):
	mkdir -p $(LOCALBIN_APP)

# Legacy support - point to new locations
BIN_DIR ?= $(LOCALBIN_APP)

# Ensure other directories exist
$(shell mkdir -p $(BUILD_DIR) $(TMP_DIR))

# ==================== Build Configuration ====================
# Go build configuration
GO := $(shell which go)
GOFLAGS := -trimpath
GOARCH ?= $(ARCH)
GOOS ?= $(OS)

# Binary names
API_SERVER_BIN := $(LOCALBIN_APP)/api-server
CLI_BIN := $(LOCALBIN_APP)/testkube
KUBECTL_TESTKUBE_CLI_BIN := $(LOCALBIN_APP)/kubectl-testkube
TOOLKIT_BIN := $(LOCALBIN_APP)/testworkflow-toolkit
INIT_BIN := $(LOCALBIN_APP)/testworkflow-init

# Docker configuration
DOCKER := docker
DOCKER_REGISTRY ?= docker.io/kubeshop

# ==================== Development ====================
# Namespace in which to deploy the sandboxed agent for development (see 'make devbox' target)
DEVBOX_NAMESPACE ?= devbox

# ==================== External Tool Versions ====================
SWAGGER_CODEGEN_VERSION := latest
GOTESTSUM_VERSION := v1.12.3
GORELEASER_VERSION := v2.11.0
GOLANGCI_LINT_VERSION := v2.5.0
SQLC_VERSION := v1.29.0

# Tool binaries
GOTESTSUM = go run gotest.tools/gotestsum@$(GOTESTSUM_VERSION)
GORELEASER = go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)
GOLANGCI_LINT = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
SQLC = go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
# swagger-codegen is installed globally via brew/package manager
SWAGGER_CODEGEN = $(shell command -v swagger-codegen 2> /dev/null)

# ==================== Environment Configuration ====================
DASHBOARD_URI ?= https://demo.testkube.io
BUSYBOX_IMAGE ?= busybox:latest
# Slack bot
SLACK_BOT_CLIENT_ID ?=
SLACK_BOT_CLIENT_SECRET ?=
# Analytics
TESTKUBE_ANALYTICS_ENABLED ?= false
ANALYTICS_TRACKING_ID ?=
ANALYTICS_API_KEY ?=
# Storage configuration
ROOT_MINIO_USER ?= minio99
ROOT_MINIO_PASSWORD ?= minio123

# ==================== Linker Flags ====================
# Common linker flags for all builds
LD_FLAGS_COMMON := -s -w \
    -X main.version=$(VERSION) \
    -X main.commit=$(COMMIT) \
    -X main.date=$(DATE) \
    -X main.builtBy=$(USER)

# API-specific linker flags
LD_FLAGS_API := $(LD_FLAGS_COMMON) \
    -X github.com/kubeshop/testkube/internal/pkg/api.Version=$(VERSION) \
    -X github.com/kubeshop/testkube/internal/pkg/api.Commit=$(COMMIT) \
    -X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientID=$(SLACK_BOT_CLIENT_ID) \
    -X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientSecret=$(SLACK_BOT_CLIENT_SECRET) \
    -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementID=$(ANALYTICS_TRACKING_ID) \
    -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementSecret=$(ANALYTICS_API_KEY) \
    -X github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants.DefaultImage=$(BUSYBOX_IMAGE)

# ==================== Help Target ====================
# Default target shows help
.DEFAULT_GOAL := help

# Help command with automatic documentation extraction
.PHONY: help
help: ## Show this help message
	@echo "Testkube Makefile - Build System"
	@echo "================================"
	@echo ""
	@echo "Usage: make [target] [VAR=value ...]"
	@echo ""
	@echo "Detected Configuration:"
	@echo "  OS:           $(OS)"
	@echo "  Architecture: $(ARCH)"
	@echo "  Go Version:   run 'go version' to check"
	@echo ""
	@echo "Available Targets by Category:"
	@awk 'BEGIN {FS = ":.*##"; current_group = ""} \
		/^##@/ { \
			group = substr($$0, 5); \
			gsub(/^[ \t]+|[ \t]+$$/, "", group); \
			if (group != current_group) { \
				current_group = group; \
				printf "\n\033[1m%s\033[0m\n", group; \
			} \
		} \
		/^[a-zA-Z_-]+:.*?##/ { \
			target = $$1; \
			desc = $$2; \
			gsub(/^[ \t]+|[ \t]+$$/, "", desc); \
			printf "  \033[36m%-28s\033[0m %s\n", target, desc; \
		}' $(MAKEFILE_LIST)

# ==================== Quick Start ====================
##@ Quick Start

.PHONY: all
all: clean build test ## Clean, build, and test everything

# ==================== Release  ====================
##@ Release

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

# ==================== Primary Build Targets ====================
##@ Build

.PHONY: build
build: build-api-server build-testkube-cli build-toolkit build-init ## Build all binaries

.PHONY: build-api-server
build-api-server: ## Build API server binary
	@echo "Building API server ($(GOOS)/$(GOARCH))..."
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		$(GOFLAGS) \
		-ldflags='$(LD_FLAGS_API)' \
		-o $(API_SERVER_BIN) \
		./cmd/api-server/
	@echo "API server built: $(API_SERVER_BIN)"

.PHONY: build-testkube-cli
build-testkube-cli: $(CLI_BIN) ## Build CLI binary (testkube)
$(CLI_BIN): $(LOCALBIN_APP)
	@echo "Building testkube CLI ($(GOOS)/$(GOARCH))..."
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		$(GOFLAGS) \
		-ldflags='$(LD_FLAGS_API)' \
		-o $(CLI_BIN) \
		cmd/kubectl-testkube/main.go
	@echo "testkube CLI built: $(CLI_BIN)"

.PHONY: build-kubectl-testkube-cli
build-kubectl-testkube-cli: $(KUBECTL_TESTKUBE_CLI_BIN) ## Build CLI binary (kubectl-testkube)
$(KUBECTL_TESTKUBE_CLI_BIN): $(LOCALBIN_APP)
	@echo "Building kubectl-testkube CLI ($(GOOS)/$(GOARCH))..."
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		$(GOFLAGS) \
		-ldflags='$(LD_FLAGS_API)' \
		-o $(KUBECTL_TESTKUBE_CLI_BIN) \
		cmd/kubectl-testkube/main.go
	@echo "kubectl-testkube CLI built: $(KUBECTL_TESTKUBE_CLI_BIN)"

.PHONY: rebuild-kubectl-testkube-cli
rebuild-kubectl-testkube-cli: ## Delete and rebuild kubectl-testkube CLI binary
	@echo "Removing existing kubectl-testkube CLI binary..."
	@rm -f $(KUBECTL_TESTKUBE_CLI_BIN)
	@$(MAKE) build-kubectl-testkube-cli

.PHONY: build-toolkit
build-toolkit: ## Build testworkflow toolkit
	@echo "Building testworkflow toolkit..."
	@CGO_ENABLED=0 $(GO) build \
		$(GOFLAGS) \
		-ldflags='$(LD_FLAGS_API)' \
		-o $(TOOLKIT_BIN) \
		cmd/testworkflow-toolkit/main.go
	@echo "Toolkit built: $(TOOLKIT_BIN)"

.PHONY: build-init
build-init: ## Build testworkflow init
	@echo "Building testworkflow init..."
	@CGO_ENABLED=0 $(GO) build \
		$(GOFLAGS) \
		-ldflags='$(LD_FLAGS_API)' \
		-o $(INIT_BIN) \
		cmd/testworkflow-init/main.go
	@echo "Init built: $(INIT_BIN)"

.PHONY: build-all-platforms
build-all-platforms: ## Build binaries for all supported platforms
	@echo "Building for all platforms..."
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			echo "Building for $$os/$$arch..."; \
			$(MAKE) build GOOS=$$os GOARCH=$$arch BIN_DIR=$(DIST_DIR)/$$os-$$arch; \
		done; \
	done

# ==================== Development ====================
##@ Development

.PHONY: run-api
run-api: ## Run API server locally
	@echo "Starting API server..."
	$(GO) run -ldflags='$(LD_FLAGS_API)' ./cmd/api-server/

.PHONY: run-api-race
run-api-race: ## Run API server with race detector
	@echo "Starting API server with race detector..."
	$(GO) run -race -ldflags='$(LD_FLAGS_API)' cmd/api-server/main.go

.PHONY: run-mongo
run-mongo: ## Run MongoDB in Docker for development (detached)
	@echo "Starting MongoDB container..."
	@$(DOCKER) run --name mongodb -p 27017:27017 --rm --detach mongo

.PHONY: run-nats
run-nats: ## Run NATS server in Docker for development (detached)
	@echo "Starting NATS server container..."
	@$(DOCKER) run --name nats -p 4222:4222 --rm --detach nats:latest

.PHONY: stop-mongo
stop-mongo: ## Stop MongoDB Docker container
	@echo "Stopping MongoDB container..."
	@$(DOCKER) stop mongodb || true

.PHONY: stop-nats
stop-nats: ## Stop NATS Docker container
	@echo "Stopping NATS server container..."
	@$(DOCKER) stop nats || true

.PHONY: login-local
login-local: $(CLI_BIN) ## Login to local Control Plane instance for CLI operations
	@echo "Logging in to local Control Plane instance..."
	@$(CLI_BIN) login --api-uri-override=http://localhost:8099 --agent-uri-override=http://testkube-enterprise-api.tk-dev.svc.cluster.local:8089 --auth-uri-override=http://localhost:5556 --custom-auth

.PHONY: devbox
devbox: $(CLI_BIN) ## Start development environment using devbox (Control Plane needs to be running and also you need to be logged in via CLI, see 'make login-local' target)
	@echo "Starting development agent with in $${DEVBOX_NAMESPACE} namespace..."
	@$(CLI_BIN) devbox --namespace $${DEVBOX_NAMESPACE}

.PHONY: dev
dev: run-mongo run-nats run-api ## Start development environment

# ==================== Testing ====================
##@ Testing

.PHONY: test
test: unit-tests ## Run all tests

.PHONY: unit-tests
unit-tests: ## Run unit tests with coverage
	@echo "Running unit tests..."
	@$(GOTESTSUM) --format short-verbose --junitfile unit-tests.xml --jsonfile unit-tests.json -- \
		-coverprofile=coverage.out -covermode=atomic ./cmd/... ./internal/... ./pkg/...

.PHONY: integration-tests
integration-tests: build-init build-toolkit ## Run integration tests (only tests ending with _Integration)
	@echo "Running integration tests (only tests ending with _Integration)..."
	@INTEGRATION="true" \
		TESTKUBE_PROJECT_ROOT="$(PWD)" \
		STORAGE_ACCESSKEYID=$(ROOT_MINIO_USER) \
		STORAGE_SECRETACCESSKEY=$(ROOT_MINIO_PASSWORD) \
		$(GOTESTSUM) --format short-verbose --junitfile integration-tests.xml --jsonfile integration-tests.json -- \
		-coverprofile=integration-coverage.out -covermode=atomic -run "_Integration$$" ./internal/... ./pkg/... ./test/integration/components/...

.PHONY: cover
cover: unit-tests ## Generate and open test coverage report
	@echo "Generating coverage report..."
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@$(OPEN_CMD) coverage.html

# ==================== Linting ====================
##@ Linting

.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@$(GOLANGCI_LINT) run ./... --timeout 10m

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with automatic fixes
	@echo "Running golangci-lint with fixes..."
	@$(GOLANGCI_LINT) run ./... --timeout 10m --fix

# ==================== Code Generation ====================
##@ Code Generation

.PHONY: generate
generate: generate-protobuf generate-openapi generate-mocks generate-sqlc generate-crds ## Generate all code

.PHONY: generate-protobuf
generate-protobuf: ## Generate protobuf code
	@echo "Generating protobuf code..."
	@go generate ./proto

.PHONY: generate-openapi
generate-openapi: swagger-codegen-check ## Generate OpenAPI models
	@echo "Generating OpenAPI models..."
	@$(SWAGGER_CODEGEN) generate --model-package testkube \
		-i api/v1/testkube.yaml -l go -o $(TMP_DIR)/api/testkube
	@bash scripts/openapi-postprocess.sh
	@$(GO) fmt pkg/api/v1/testkube/*.go

.PHONY: generate-mocks
generate-mocks: ## Generate mock files using mockgen only in ./cmd, ./internal, and ./pkg
	@echo "Generating mock files..."
	@go generate -run mockgen -x ./...

.PHONY: generate-sqlc
generate-sqlc: ## Generate sqlc package with sql queries
	@echo "Generating sqlc queries..."
	@$(SQLC) generate

.PHONY: generate-crds
generate-crds: ## Generate Kubernetes CRDs from kubebuilder Golang structs.
	# Generate CRDs
	go tool controller-gen crd:allowDangerousTypes=true object paths="./api/..." output:crd:dir=k8s/crd

    # Reduce size of TestWorkflow CRDs to fit in the "last-applied" annotation which has a limit of 262144 bytes.
	@for file in testworkflows.testkube.io_testworkflows.yaml testworkflows.testkube.io_testworkflowtemplates.yaml testworkflows.testkube.io_testworkflowexecutions.yaml; do \
		for key in securityContext volumes dnsPolicy affinity tolerations hostAliases dnsConfig topologySpreadConstraints schedulingGates resourceClaims imagePullSecrets volumeMounts fieldRef resourceFieldRef configMapKeyRef secretKeyRef pvcs matchExpressions matchLabels env envFrom fileKeyRef readinessProbe; do \
			go tool yq --no-colors -i "del(.. | select(has(\"$$key\")).$$key | .. | select(has(\"description\")).description)" "k8s/crd/$$file"; \
		done; \
		go tool yq --no-colors -i \
		'with(..; . | select(has("additionalProperties")) | select(.additionalProperties | has("type")) | select(.additionalProperties.type == "dynamicList") | \
			.["x-kubernetes-preserve-unknown-fields"] = true | \
			del(.additionalProperties) \
		) | \
		with(..; . | select(has("properties")) | select(.properties | to_entries | filter(.value | has("type")) | filter(.value.type == "dynamicList") | length > 0) | \
			.["x-kubernetes-preserve-unknown-fields"] = true | \
			del(.properties) \
		)' \
		"k8s/crd/$$file"; \
	done

	# Copy to testkube-operator chart as Helm Templated
	node js/scripts/crd-postprocess.js

# ==================== Docker ====================
##@ Docker

.PHONY: docker-build
docker-build: docker-build-api docker-build-cli ## Build all Docker images

.PHONY: docker-build-api
docker-build-api: ## Build API server Docker image
	@echo "Building API server Docker image..."
	@env SLACK_BOT_CLIENT_ID=** SLACK_BOT_CLIENT_SECRET=** \
		ANALYTICS_TRACKING_ID=** ANALYTICS_API_KEY=** \
		SEGMENTIO_KEY=** CLOUD_SEGMENTIO_KEY=** \
		DOCKER_BUILDX_CACHE_FROM=type=registry,ref=$(DOCKER_REGISTRY)/testkube-api-server:latest \
		ALPINE_IMAGE=alpine:3.20.6 \
		$(GORELEASER) release -f goreleaser_files/.goreleaser-docker-build-api.yml --clean --snapshot

.PHONY: docker-build-cli
docker-build-cli: ## Build CLI Docker image
	@echo "Building CLI Docker image..."
	@env SLACK_BOT_CLIENT_ID=** SLACK_BOT_CLIENT_SECRET=** \
		ANALYTICS_TRACKING_ID=** ANALYTICS_API_KEY=** \
		SEGMENTIO_KEY=** CLOUD_SEGMENTIO_KEY=** \
		DOCKER_BUILDX_CACHE_FROM=type=registry,ref=$(DOCKER_REGISTRY)/testkube-cli:latest \
		ALPINE_IMAGE=alpine:3.20.6 \
		$(GORELEASER) release -f .builds-linux.goreleaser.yml --clean --snapshot

# ==================== Kubernetes ====================
##@ Kubernetes

.PHONY: port-forward-api
port-forward-api: ## Port forward to API server
	@echo "Port forwarding to API server..."
	@kubectl port-forward svc/testkube-api-server 8088 -n$(NAMESPACE)

.PHONY: port-forward-mongo
port-forward-mongo: ## Port forward to MongoDB
	@echo "Port forwarding to MongoDB..."
	@kubectl port-forward svc/testkube-mongodb 27017 -n$(NAMESPACE)

.PHONY: port-forward-minio
port-forward-minio: ## Port forward to MinIO
	@echo "Port forwarding to MinIO..."
	@kubectl port-forward svc/testkube-minio-service-testkube 9090:9090 -n$(NAMESPACE)

# ==================== Documentation ====================
##@ Documentation

.PHONY: docs
docs: commands-reference ## Generate documentation

.PHONY: commands-reference
commands-reference: ## Generate CLI command reference
	@echo "Generating command reference..."
	@mkdir -p gen/docs/cli
	@$(GO) run cmd/kubectl-testkube/main.go generate doc

# ==================== Maintenance ====================
##@ Maintenance

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR) $(TMP_DIR)
	@rm -f coverage.html coverage.out integration-coverage.out
	@rm -f unit-tests.xml unit-tests.json
	@rm -f integration-tests.xml integration-tests.json
	@echo "Clean complete"

.PHONY: clean-all
clean-all: clean ## Deep clean including Go cache
	@echo "Performing deep clean..."
	@go clean -cache -testcache -modcache
	@echo "Deep clean complete"

# ==================== Tool Installation ====================
##@ Tools

# Tool installation targets
.PHONY: swagger-codegen-check
swagger-codegen-check: ## Check if swagger-codegen is installed
ifndef SWAGGER_CODEGEN
	$(error swagger-codegen is not installed. Please install it manually from https://github.com/swagger-api/swagger-codegen)
endif

# ==================== Utility Functions ====================
# Color output helpers
define print_info
	@printf "\033[36m%s\033[0m\n" "$(1)"
endef

define print_success
	@printf "\033[32m✓ %s\033[0m\n" "$(1)"
endef

define print_error
	@printf "\033[31m✗ %s\033[0m\n" "$(1)"
endef

# ==================== Special/Experimental ====================
##@ Special

.PHONY: video
video: ## Generate project activity video using gource
	@echo "Generating project activity video..."
	@gource \
		-s .5 \
		-1280x720 \
		--auto-skip-seconds .1 \
		--multi-sampling \
		--stop-at-end \
		--key \
		--highlight-users \
		--date-format "%d/%m/%y" \
		--hide mouse,filenames \
		--file-idle-time 0 \
		--max-files 0 \
		--background-colour 000000 \
		--font-size 25 \
		--output-ppm-stream stream.out \
		--output-framerate 30
	@ffmpeg -y -r 30 -f image2pipe -vcodec ppm -i stream.out -b 65536K movie.mp4
	@rm stream.out
