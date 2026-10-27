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

# Run targets one at a time, even when make is invoked with -j
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

# Legacy variable: no build recipe reads it (build-all-platforms still passes
# it), so binaries always go to LOCALBIN_APP
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
CONVERT_BIN := $(LOCALBIN_APP)/convert

# Docker configuration
DOCKER := docker
DOCKER_REGISTRY ?= docker.io/kubeshop

# ==================== Development ====================
# Namespace in which to deploy the sandboxed agent for development (see 'make devbox' target)
DEVBOX_NAMESPACE ?= devbox

# ==================== External Tool Versions ====================
GOTESTSUM_VERSION := v1.12.3
GORELEASER_VERSION := v2.11.0
GOLANGCI_LINT_VERSION := v2.15.0
SQLC_VERSION := v1.30.0
BUF_VERSION := v1.68.1

# Tool binaries
GOTESTSUM = go run gotest.tools/gotestsum@$(GOTESTSUM_VERSION)
GORELEASER = go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)
GOLANGCI_LINT = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
SQLC = go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
BUF = $(LOCALBIN)/tooling/buf

# ==================== Environment Configuration ====================
DASHBOARD_URI ?= https://demo.testkube.io
BUSYBOX_IMAGE ?= busybox:latest
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
		/^[a-zA-Z0-9_%\/-][a-zA-Z0-9_.%\/-]*:.*##/ { \
			target = $$1; \
			desc = $$2; \
			gsub(/^[ \t]+|[ \t]+$$/, "", desc); \
			printf "  \033[36m%-28s\033[0m %s\n", target, desc; \
		}' $(MAKEFILE_LIST)

# ==================== Quick Start ====================
##@ Quick Start

.PHONY: all
all: clean build test ## Run clean, build and test (clean also deletes the tracked build/ directory)

# ==================== Release  ====================
##@ Release
# Each bump creates a v-prefixed git tag from the newest existing tag and runs
# 'git push --tags' (pushes every local tag to the default remote).

.PHONY: version-bump
version-bump: version-bump-patch ## Same as version-bump-patch

.PHONY: version-bump-patch
version-bump-patch: ## Tag the next patch version and push all tags
	go run cmd/tools/main.go bump -k patch

.PHONY: version-bump-minor
version-bump-minor: ## Tag the next minor version and push all tags
	go run cmd/tools/main.go bump -k minor

.PHONY: version-bump-major
version-bump-major: ## Tag the next major version and push all tags
	go run cmd/tools/main.go bump -k major

.PHONY: version-bump-dev
version-bump-dev: ## Tag the next -betaNNN prerelease and push all tags
	go run cmd/tools/main.go bump --dev

# ==================== Primary Build Targets ====================
##@ Build

.PHONY: build
build: build-api-server build-testkube-cli build-toolkit build-init build-convert ## Build api-server, testkube CLI, toolkit, init and convert into bin/app

.PHONY: compile
compile: ## Check that every package compiles (go build ./...), writing no binaries
	@echo "Compiling all packages..."
	@$(GO) build ./...

.PHONY: build-api-server
build-api-server: ## Build the API server binary (bin/app/api-server)
	@echo "Building API server ($(GOOS)/$(GOARCH))..."
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		$(GOFLAGS) \
		-ldflags='$(LD_FLAGS_API)' \
		-o $(API_SERVER_BIN) \
		./cmd/api-server/
	@echo "API server built: $(API_SERVER_BIN)"

.PHONY: build-testkube-cli $(CLI_BIN)
build-testkube-cli: $(CLI_BIN) ## Build the CLI as bin/app/testkube (rebuilds every time)
$(CLI_BIN): $(LOCALBIN_APP)
	@echo "Building testkube CLI ($(GOOS)/$(GOARCH))..."
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		$(GOFLAGS) \
		-ldflags='$(LD_FLAGS_API)' \
		-o $(CLI_BIN) \
		cmd/kubectl-testkube/main.go
	@echo "testkube CLI built: $(CLI_BIN)"

.PHONY: build-kubectl-testkube-cli $(KUBECTL_TESTKUBE_CLI_BIN)
build-kubectl-testkube-cli: $(KUBECTL_TESTKUBE_CLI_BIN) ## Build the CLI as bin/app/kubectl-testkube (rebuilds every time)
$(KUBECTL_TESTKUBE_CLI_BIN): $(LOCALBIN_APP)
	@echo "Building kubectl-testkube CLI ($(GOOS)/$(GOARCH))..."
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		$(GOFLAGS) \
		-ldflags='$(LD_FLAGS_API)' \
		-o $(KUBECTL_TESTKUBE_CLI_BIN) \
		cmd/kubectl-testkube/main.go
	@echo "kubectl-testkube CLI built: $(KUBECTL_TESTKUBE_CLI_BIN)"

.PHONY: rebuild-kubectl-testkube-cli
rebuild-kubectl-testkube-cli: ## Delete bin/app/kubectl-testkube and build it again
	@echo "Removing existing kubectl-testkube CLI binary..."
	@rm -f $(KUBECTL_TESTKUBE_CLI_BIN)
	@$(MAKE) build-kubectl-testkube-cli

.PHONY: build-toolkit
build-toolkit: ## Build the testworkflow-toolkit binary (bin/app/testworkflow-toolkit)
	@echo "Building testworkflow toolkit..."
	@CGO_ENABLED=0 $(GO) build \
		$(GOFLAGS) \
		-ldflags='$(LD_FLAGS_API)' \
		-o $(TOOLKIT_BIN) \
		cmd/testworkflow-toolkit/main.go
	@echo "Toolkit built: $(TOOLKIT_BIN)"

.PHONY: build-init
build-init: ## Build the testworkflow-init binary (bin/app/testworkflow-init)
	@echo "Building testworkflow init..."
	@CGO_ENABLED=0 $(GO) build \
		$(GOFLAGS) \
		-ldflags='$(LD_FLAGS_API)' \
		-o $(INIT_BIN) \
		cmd/testworkflow-init/main.go
	@echo "Init built: $(INIT_BIN)"

.PHONY: build-convert
build-convert: ## Build the Mongo -> Postgres convert tool (bin/app/convert)
	@echo "Building convert tool ($(GOOS)/$(GOARCH))..."
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		$(GOFLAGS) \
		-ldflags='-s -w -X main.commit=$(COMMIT) -X main.date=$(DATE) -X main.builtBy=$(USER) -X github.com/kubeshop/testkube/pkg/version.Version=$(VERSION) -X github.com/kubeshop/testkube/pkg/version.Commit=$(COMMIT)' \
		-o $(CONVERT_BIN) \
		./cmd/convert/
	@echo "Convert tool built: $(CONVERT_BIN)"

.PHONY: build-all-platforms
build-all-platforms: ## Run build for linux/darwin/windows on amd64/arm64; each pass overwrites bin/app, so only the last platform's binaries remain
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
run-api: ## Run the API server from source (go run)
	@echo "Starting API server..."
	$(GO) run -ldflags='$(LD_FLAGS_API)' ./cmd/api-server/

.PHONY: run-api-race
run-api-race: ## Run the API server from source with the race detector
	@echo "Starting API server with race detector..."
	$(GO) run -race -ldflags='$(LD_FLAGS_API)' cmd/api-server/main.go

.PHONY: run-mongo
run-mongo: ## Start a detached MongoDB container named mongodb on port 27017 (needs Docker)
	@echo "Starting MongoDB container..."
	@$(DOCKER) run --name mongodb -p 27017:27017 --rm --detach mongo

.PHONY: run-nats
run-nats: ## Start a detached NATS container named nats on port 4222 (needs Docker)
	@echo "Starting NATS server container..."
	@$(DOCKER) run --name nats -p 4222:4222 --rm --detach nats:latest

.PHONY: stop-mongo
stop-mongo: ## Stop the mongodb container started by run-mongo
	@echo "Stopping MongoDB container..."
	@$(DOCKER) stop mongodb || true

.PHONY: stop-nats
stop-nats: ## Stop the nats container started by run-nats
	@echo "Stopping NATS server container..."
	@$(DOCKER) stop nats || true

.PHONY: login-local
login-local: $(CLI_BIN) ## Build the CLI and log it in to a local Control Plane (localhost:8099, tk-dev namespace)
	@echo "Logging in to local Control Plane instance..."
	@$(CLI_BIN) login --api-uri-override=http://localhost:8099 --agent-uri-override=http://testkube-enterprise-api.tk-dev.svc.cluster.local:8089 --auth-uri-override=http://localhost:5556 --custom-auth

.PHONY: devbox
devbox: $(CLI_BIN) ## Build the CLI and start a devbox agent in DEVBOX_NAMESPACE (needs a running Control Plane and 'make login-local' first)
	@echo "Starting development agent with in $${DEVBOX_NAMESPACE} namespace..."
	@$(CLI_BIN) devbox --namespace $${DEVBOX_NAMESPACE}

.PHONY: dev
dev: run-mongo run-nats run-api ## Start MongoDB and NATS containers, then run the API server (needs Docker)

# ==================== Testing ====================
##@ Testing

.PHONY: test
test: unit-tests ## Run the unit tests (same as unit-tests)

# Narrow unit-tests and integration-tests to some packages and tests, e.g.
#   make unit-tests PKG=./pkg/mcp/... RUN='^TestName$'
# RUN is a go test -run pattern; integration-tests defaults it to _Integration$.
PKG ?=
RUN ?=

.PHONY: unit-tests
unit-tests: ## Run unit tests (narrow with PKG=, RUN=); writes unit-tests.xml, unit-tests.json and coverage.out
	@echo "Running unit tests..."
	@$(GOTESTSUM) --format short-verbose --junitfile unit-tests.xml --jsonfile unit-tests.json -- \
		-coverprofile=coverage.out -covermode=atomic $(if $(value RUN),-run '$(value RUN)') \
		$(or $(PKG),./cmd/... ./internal/... ./pkg/...)

.PHONY: integration-tests
integration-tests: build-init build-toolkit ## Run *_Integration tests against their backing services (e.g. Postgres, MinIO; narrow with PKG=, RUN=); writes integration-tests.xml/.json and integration-coverage.out
	@echo "Running integration tests (only tests ending with _Integration)..."
	@INTEGRATION="true" \
		TESTKUBE_PROJECT_ROOT="$(PWD)" \
		STORAGE_ACCESSKEYID=$(ROOT_MINIO_USER) \
		STORAGE_SECRETACCESSKEY=$(ROOT_MINIO_PASSWORD) \
		$(GOTESTSUM) --format short-verbose --junitfile integration-tests.xml --jsonfile integration-tests.json -- \
		-coverprofile=integration-coverage.out -covermode=atomic -run '$(or $(value RUN),_Integration$$)' \
		$(or $(PKG),./internal/... ./pkg/... ./test/integration/components/...)

.PHONY: cover
cover: unit-tests ## Run unit tests, write coverage.html and open it in a browser
	@echo "Generating coverage report..."
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@$(OPEN_CMD) coverage.html

.PHONY: bench
bench: ## Run benchmarks that need no external services
	@echo "Running benchmarks..."
	@$(GO) test -run '^$$' -bench . -benchmem ./internal/... ./pkg/...

.PHONY: bench-integration
bench-integration: ## Run benchmarks that need Postgres (same env as integration-tests)
	@echo "Running integration benchmarks..."
	@INTEGRATION="true" \
		TESTKUBE_PROJECT_ROOT="$(PWD)" \
		$(GO) test -run '^$$' -bench '_Integration$$' -benchmem -benchtime 10x ./internal/... ./pkg/...

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

.PHONY: security-scan
security-scan: ## Run the PR security scan (Semgrep, Gitleaks, Trivy) on your changes vs origin/main. Needs Docker; ENFORCE=1 fails on findings
	@./scripts/security-scan.sh

# ==================== Code Generation ====================
##@ Code Generation

.PHONY: generate
generate: generate-protobuf generate-openapi generate-mocks generate-sqlc generate-crds ## Run all generators: protobuf, OpenAPI, mocks, sqlc and CRDs (needs Docker and node)

.PHONY: generate-protobuf
generate-protobuf: $(BUF) ## Regenerate protobuf code from proto/ with buf
	@echo "Generating protobuf code..."
	@cd proto && $(abspath $(BUF)) generate --exclude-path service.proto --exclude-path logs.proto
	@cd proto && $(abspath $(BUF)) generate --template buf.gen.old.yaml

.PHONY: generate-openapi
generate-openapi: ## Regenerate OpenAPI models from api/v1/testkube.yaml (needs Docker)
	@bash scripts/generate-openapi.sh

.PHONY: generate-mocks
generate-mocks: ## Regenerate mocks (go generate -run mockgen ./...)
	@echo "Generating mock files..."
	@go generate -run mockgen -x ./...

.PHONY: generate-sqlc
generate-sqlc: ## Regenerate sqlc query code
	@echo "Generating sqlc queries..."
	@$(SQLC) generate

.PHONY: generate-crds
generate-crds: ## Regenerate CRDs in k8s/crd, the Helm CRD templates and vendored Helm libraries (needs node)
	# Generate CRDs
	go tool controller-gen crd:allowDangerousTypes=true object paths="./api/..." output:crd:dir=k8s/crd

	# Reduce size of TestWorkflow CRDs to fit in the "last-applied" annotation which has a limit of 262144 bytes.
	@for file in testworkflows.testkube.io_testworkflows.yaml testworkflows.testkube.io_testworkflowtemplates.yaml testworkflows.testkube.io_testworkflowexecutions.yaml; do \
		for key in securityContext volumes dnsPolicy affinity tolerations hostAliases dnsConfig topologySpreadConstraints schedulingGates resourceClaims imagePullSecrets volumeMounts fieldRef resourceFieldRef configMapKeyRef secretKeyRef pvcs matchExpressions matchLabels env envFrom fileKeyRef readinessProbe; do \
			go tool yq --no-colors -i "del(.. | select(has(\"$$key\")).$$key | .. | select(has(\"description\")).description)" "k8s/crd/$$file"; \
		done; \
		go tool yq --no-colors -i 'with(..; . | select(has("additionalProperties")) | select(.additionalProperties | has("type")) | select(.additionalProperties.type == "dynamicList") | .["x-kubernetes-preserve-unknown-fields"] = true | del(.additionalProperties)) | with(..; . | select(has("properties")) | select(.properties | to_entries | filter(.value | has("type")) | filter(.value.type == "dynamicList") | length > 0) | .["x-kubernetes-preserve-unknown-fields"] = true | del(.properties))' \
		"k8s/crd/$$file"; \
	done

	# Copy to shared Helm library chart as templated CRDs
	node js/scripts/crd-postprocess.js

	# Sync shared Helm library charts into GitOps-rendered consumers
	bash scripts/sync-helm-libraries.sh

.PHONY: sync-helm-libraries
sync-helm-libraries: ## Sync shared Helm library charts into consumer chart dependencies
	bash scripts/sync-helm-libraries.sh

.PHONY: verify-helm-libraries-synced
verify-helm-libraries-synced: ## Fail if vendored Helm library charts differ from the canonical sources
	bash scripts/check-helm-libraries.sh

.PHONY: verify-protobuf-generated
verify-protobuf-generated: generate-protobuf ## Regenerate protobuf code and fail if that changes tracked files
	git diff --exit-code -- pkg/proto/ pkg/cloud/ pkg/logs/pb/

.PHONY: verify-crds-generated
verify-crds-generated: generate-crds verify-helm-libraries-synced ## Regenerate CRDs, check Helm libraries, and fail if that changes tracked files
	git diff --exit-code -- k8s/crd k8s/helm/testkube-crds/templates/_generated_crds.tpl k8s/helm/testkube-operator/charts/testkube-crds/templates/_generated_crds.tpl k8s/helm/testkube-runner/charts/testkube-crds/templates/_generated_crds.tpl

# ==================== Docker ====================
##@ Docker

.PHONY: docker-build
docker-build: docker-build-api docker-build-cli ## Run docker-build-api and docker-build-cli

.PHONY: docker-build-api
docker-build-api: ## Build the API server images for linux/amd64 and linux/arm64/v8 with goreleaser --snapshot (needs Docker buildx able to build both platforms)
	@echo "Building API server Docker image..."
	@env ANALYTICS_TRACKING_ID=** ANALYTICS_API_KEY=** \
		SEGMENTIO_KEY=** CLOUD_SEGMENTIO_KEY=** \
		DOCKER_BUILDX_CACHE_FROM=type=registry,ref=$(DOCKER_REGISTRY)/testkube-api-server:latest \
		ALPINE_IMAGE=alpine:3.24.2 \
		$(GORELEASER) release -f goreleaser_files/.goreleaser-docker-build-api.yml --clean --snapshot

.PHONY: docker-build-cli
docker-build-cli: ## Build linux kubectl-testkube binaries into linux/ with goreleaser --snapshot; no image is built (needs KEYGEN_PUBLIC_KEY set)
	@echo "Building CLI Docker image..."
	@env ANALYTICS_TRACKING_ID=** ANALYTICS_API_KEY=** \
		SEGMENTIO_KEY=** CLOUD_SEGMENTIO_KEY=** \
		DOCKER_BUILDX_CACHE_FROM=type=registry,ref=$(DOCKER_REGISTRY)/testkube-cli:latest \
		ALPINE_IMAGE=alpine:3.24.2 \
		$(GORELEASER) release -f .builds-linux.goreleaser.yml --clean --snapshot

# ==================== Kubernetes ====================
##@ Kubernetes

.PHONY: port-forward-api
port-forward-api: ## Forward port 8088 to svc/testkube-api-server in NAMESPACE (needs a cluster)
	@echo "Port forwarding to API server..."
	@kubectl port-forward svc/testkube-api-server 8088 -n$(NAMESPACE)

.PHONY: port-forward-mongo
port-forward-mongo: ## Forward port 27017 to svc/testkube-mongodb in NAMESPACE (needs a cluster)
	@echo "Port forwarding to MongoDB..."
	@kubectl port-forward svc/testkube-mongodb 27017 -n$(NAMESPACE)

.PHONY: port-forward-minio
port-forward-minio: ## Forward port 9090 to svc/testkube-minio-service-testkube in NAMESPACE (needs a cluster)
	@echo "Port forwarding to MinIO..."
	@kubectl port-forward svc/testkube-minio-service-testkube 9090:9090 -n$(NAMESPACE)

# ==================== Documentation ====================
##@ Documentation

.PHONY: docs
docs: commands-reference ## Generate the CLI reference (same as commands-reference)

.PHONY: commands-reference
commands-reference: ## Generate the CLI command reference as Markdown in gen/docs/cli
	@echo "Generating command reference..."
	@mkdir -p gen/docs/cli
	@$(GO) run cmd/kubectl-testkube/main.go generate doc

# ==================== Maintenance ====================
##@ Maintenance

.PHONY: clean
clean: ## Delete build/ (including its tracked files), dist/, tmp/ and test and coverage outputs
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR) $(TMP_DIR)
	@rm -f coverage.html coverage.out integration-coverage.out
	@rm -f unit-tests.xml unit-tests.json
	@rm -f integration-tests.xml integration-tests.json
	@echo "Clean complete"

.PHONY: clean-all
clean-all: clean ## Run clean, then clear the Go build, test and module caches
	@echo "Performing deep clean..."
	@go clean -cache -testcache -modcache
	@echo "Deep clean complete"

# ==================== Tool Installation ====================
##@ Tools

.PHONY: buf
buf: $(BUF) ## Download buf to bin/tooling/buf (used by generate-protobuf)
$(BUF):
	@mkdir -p $(dir $(BUF))
	@echo "Downloading buf $(BUF_VERSION)..."
	@curl --fail -sSL "https://github.com/bufbuild/buf/releases/download/$(BUF_VERSION)/buf-$(shell uname -s)-$(shell uname -m)" -o $(BUF)
	@chmod +x $(BUF)
	@echo "buf $(BUF_VERSION) installed to $(BUF)"
