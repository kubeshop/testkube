# Testkube Agent Architecture (Standalone Mode)

This document describes the high-level architecture of the Testkube agent when running in standalone mode. It's intended to help new contributors understand how the system works and navigate the codebase.

> In standalone mode, the Testkube Agent manages all storage and orchestration locally without connecting to a remote control plane - [Read More](https://docs.testkube.io/articles/install/standalone-agent).

## Table of Contents

### Core Components

- [1. API Server](#1-api-server)
- [2. Kubernetes Controllers](#2-kubernetes-controllers)
- [3. Workflow Execution Runtime](#3-workflow-execution-runtime)
- [4. Storage Layer](#4-storage-layer)
- [5. Event System](#5-event-system)
- [6. REST API](#6-rest-api)
- [7. Prometheus Metrics Endpoint](#7-prometheus-metrics-endpoint)
- [8. Logging and Telemetry](#8-logging-and-telemetry)
- [9. Kubernetes Custom Resource Definitions (CRDs)](#9-kubernetes-custom-resource-definitions-crds)

### Architecture Overview

- [Data Flow: TestWorkflow Execution](#data-flow-testworkflow-execution)
- [Deployment](#deployment)
- [Key Configuration](#key-configuration)

### Client Tools

- [CLI](#cli)

### References

- [Related Documentation](#related-documentation)

## Core Components

### 1. API Server

**Entry Point**: [`cmd/api-server/main.go`](cmd/api-server/main.go)

The API server is the main service that:
- Exposes REST (HTTP) and gRPC APIs for managing tests, workflows, and executions
- Handles test/workflow execution requests
- Manages storage connections (PostgreSQL, MinIO, NATS)
- Runs Kubernetes controllers for watching CRDs
- Processes events and webhooks

**Key Packages**:
- [`internal/app/api/v1/`](internal/app/api/v1/) - HTTP/gRPC API handlers
- [`internal/config/`](internal/config/) - Configuration and environment variables
- [`pkg/server/`](pkg/server/) - HTTP/gRPC server setup

### 2. Kubernetes Controllers

**Location**: [`pkg/controller/`](pkg/controller/)

Controllers watch Kubernetes Custom Resource Definitions (CRDs) and trigger actions:
- **TestWorkflowExecution Controller** (`testworkflowexecutionexecutor.go`) - Watches `TestWorkflowExecution` CRDs and schedules workflow runs when CRDs are created/updated

Controllers are enabled via `ENABLE_K8S_CONTROLLERS=true` and use [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime).

### 3. Workflow Execution Runtime

Testkube uses [Test Workflows](https://docs.testkube.io/articles/test-workflows) as an abstraction layer for running any kind of test inside Kubernetes.

**TestWorkflow Init**: [`cmd/testworkflow-init/`](cmd/testworkflow-init/)
- Initializes TestWorkflow execution containers
- Orchestrates workflow step groups and parallel execution
- Handles container lifecycle and coordination

**TestWorkflow Toolkit**: [`cmd/testworkflow-toolkit/`](cmd/testworkflow-toolkit/)
- Runtime utilities for TestWorkflow containers
- Artifact collection and upload
- Log streaming and aggregation

**Execution Logic**: [`pkg/testworkflows/`](pkg/testworkflows/)
- Core TestWorkflow executor (`testworkflowexecutor/`)
- TestWorkflow processing and step execution
- Result aggregation and status management

### 4. Storage Layer

**PostgreSQL** (Future Primary Database, currently in Preview)
- Stores test definitions, executions, webhooks, and metadata
- Repository layer: [`pkg/repository/postgres/`](pkg/repository/postgres/)
- Migration: [`pkg/dbmigrator/`](pkg/dbmigrator/)

**MongoDB** (Current Primary Database)
- Alternative to PostgreSQL for storing test definitions, executions, webhooks, and metadata
- Repository layer: [`pkg/repository/testworkflow/mongo/`](pkg/repository/testworkflow/mongo/)
- Lease backend: [`pkg/repository/leasebackend/mongo/`](pkg/repository/leasebackend/mongo/)
- Factory: [`pkg/repository/mongo_factory.go`](pkg/repository/mongo_factory.go)

**MinIO** (Object Storage)
- Stores test artifacts (logs, reports, files)
- Buckets: `testkube-artifacts`, `testkube-logs`
- Storage interface: [`pkg/storage/`](pkg/storage/)

**NATS** (Message Queue)
- Async job processing and event publishing
- Event bus: [`pkg/event/bus/`](pkg/event/bus/)

### 5. Event System

**Location**: [`pkg/event/`](pkg/event/)

The event system publishes and listens to test execution events:
- **Event Listeners**: [`pkg/event/kind/`](pkg/event/kind/) - Webhooks, Slack, K8s events, CD events, WebSockets
- **Event Emitter**: [`pkg/event/emitter.go`](pkg/event/emitter.go) - Publishes execution lifecycle events

### 6. REST API

Testkube exposes REST APIs for interacting with core resources and functionality - [Read More](https://docs.testkube.io/openapi/overview).

**OpenAPI Definition**: [`api/v1/testkube.yaml`](api/v1/testkube.yaml)
- Defines the agent's REST API contract
- Generated models: [`pkg/api/v1/testkube/`](pkg/api/v1/testkube/)

**Framework**: Uses [Fiber](https://gofiber.io/) web framework for HTTP routing and middleware

**OpenAPI Definition**: [`api/v1/testkube.yaml`](api/v1/testkube.yaml)
- Defines the complete REST API contract
- Used for client code generation and documentation
- Generated models: [`pkg/api/v1/testkube/`](pkg/api/v1/testkube/)

**Route Registration**: [`internal/app/api/v1/server.go`](internal/app/api/v1/server.go) - `TestkubeAPI.Init()`

**Handler Implementation**: [`internal/app/api/v1/`](internal/app/api/v1/)
- Handlers: `testworkflows.go`, `testworkflowexecutions.go`, `webhook.go`, etc.
- Each handler function (e.g., `ListTestWorkflowsHandler()`) returns a Fiber handler
- Handlers interact with repositories, executors, and event emitters

**Response Formats**: Supports JSON and YAML (via `Accept` header)
- Default: `application/json`
- Alternative: `text/yaml` or `application/yaml`

**Port**: HTTP API listens on port 8088 (configurable via environment variables)

### 7. Prometheus Metrics Endpoint

**Endpoint**: `GET /metrics`

The API server exposes Prometheus metrics at `/metrics` for monitoring and observability - [Read More](https://docs.testkube.io/articles/metrics).

**Metrics Implementation**: [`internal/app/api/metrics/metrics.go`](internal/app/api/metrics/metrics.go)

**Server Setup**: The metrics endpoint is registered in [`pkg/server/httpserver.go`](pkg/server/httpserver.go) using Prometheus's standard HTTP handler (`promhttp.Handler()`).

**Access**: Metrics are accessible at `http://localhost:8088/metrics` (or the configured API server port).

### 8. Logging and Telemetry

#### Logging

**Framework**: Uses [zap](https://github.com/uber-go/zap) structured logging library

**Implementation**: [`pkg/log/log.go`](pkg/log/log.go)

**Configuration**:
- **Log Level**: Controlled via `DEBUG` environment variable
  - Default: `InfoLevel`
  - Set `DEBUG=true` for `DebugLevel`
- **Output Format**: Controlled via `LOGGER_JSON` environment variable
  - Default: Production format (JSON)
  - Set `LOGGER_JSON=true` for Development format (human-readable)

**Usage**:
- **Default Logger**: `log.DefaultLogger` - Singleton logger used throughout the codebase
- **Logger Methods**: 
  - `Info()`, `Infow()` - Information messages
  - `Debug()`, `Debugw()` - Debug messages
  - `Error()`, `Errorw()` - Error messages
  - `Warn()`, `Warnw()` - Warning messages
- **Structured Logging**: Use `Infow()`, `Errorw()`, etc. for structured logs with key-value pairs
  - Example: `log.DefaultLogger.Infow("connected to database", "host", dbHost, "port", dbPort)`

**Timestamps**: Logs include RFC3339 formatted timestamps


#### Telemetry

**Implementation**: [`pkg/telemetry/`](pkg/telemetry/)

Telemetry collects usage analytics to help improve the product. It can be disabled by users.

**Telemetry Backends**:

- **Segment.io** (`sender_sio.go`) - Primary analytics backend
- **Google Analytics** (`sender_ga4.go`) - Alternative analytics backend
- **Testkube Analytics** (`sender_tka.go`) - Internal analytics

### 9. Kubernetes Custom Resource Definitions (CRDs)

**Definition Location**: [`api/`](api/)
**Generated CRDs**: [`k8s/crd/`](k8s/crd/)

Testkube extends Kubernetes with Custom Resource Definitions to enable declarative test management. CRDs are defined using [Kubebuilder](https://book.kubebuilder.io/) annotations and generated from Go types.

**CRD Generation**: Run `make generate-crds` to regenerate CRDs after modifying types in `api/`.

> **Legacy CRDs** are no longer supported by Testkube but still included to avoid deletion of corresponding resources on deployment.

#### TestWorkflow CRDs

- **`TestWorkflow`** (`testworkflows.testkube.io/v1`)
  - **Definition**: [`api/testworkflows/v1/testworkflow_types.go`](api/testworkflows/v1/testworkflow_types.go)
  - **Purpose**: Defines a test workflow with setup, steps, and after phases
  - **Features**: Template inclusion, parallel execution, service dependencies, PVCs
  - **Status**: Tracks latest execution and health metrics

- **`TestWorkflowTemplate`** (`testworkflows.testkube.io/v1`)
  - **Definition**: [`api/testworkflows/v1/testworkflowtemplate_types.go`](api/testworkflows/v1/testworkflowtemplate_types.go)
  - **Purpose**: Reusable workflow templates with configurable parameters
  - **Usage**: Can be included in `TestWorkflow` specs via `use` field

- **`TestWorkflowExecution`** (`testworkflows.testkube.io/v1`)
  - **Definition**: [`api/testworkflows/v1/testworkflowexecution_types.go`](api/testworkflows/v1/testworkflowexecution_types.go)
  - **Purpose**: Represents an execution of a test workflow
  - **Controller**: Watched by `TestWorkflowExecutionController` (see [Kubernetes Controllers](#2-kubernetes-controllers))
  - **Status**: Tracks execution state, results, logs, and artifacts

#### Test CRDs (Legacy)

- **`Test`** (`tests.testkube.io/v1`, v2, v3)
  - **Definition**: [`api/tests/v1/`, `api/tests/v2/`, `api/tests/v3/`](api/tests/)
  - **Purpose**: Defines a test using an executor (Cypress, Postman, etc.)

- **`TestExecution`** (`tests.testkube.io/v1`)
  - **Definition**: [`api/testexecution/v1/testexecution_types.go`](api/testexecution/v1/testexecution_types.go)
  - **Purpose**: Represents an execution of a test

- **`TestSource`** (`tests.testkube.io/v1`)
  - **Definition**: [`api/testsource/v1/testsource_types.go`](api/testsource/v1/testsource_types.go)
  - **Purpose**: Defines test source code location (Git, S3, etc.)

#### TestSuite CRDs (Legacy)

- **`TestSuite`** (`tests.testkube.io/v1`, v2, v3)
  - **Definition**: [`api/testsuite/v1/`, `api/testsuite/v2/`, `api/testsuite/v3/`](api/testsuite/)
  - **Purpose**: Groups multiple tests for sequential execution

- **`TestSuiteExecution`** (`tests.testkube.io/v1`)
  - **Definition**: [`api/testsuiteexecution/v1/testsuiteexecution_types.go`](api/testsuiteexecution/v1/testsuiteexecution_types.go)
  - **Purpose**: Represents an execution of a test suite

#### Executor CRDs (Legacy)

- **`Executor`** (`executor.testkube.io/v1`)
  - **Definition**: [`api/executor/v1/executor_types.go`](api/executor/v1/executor_types.go)
  - **Purpose**: Defines test executors (Cypress, Postman, Playwright, etc.)
  - **Usage**: Specifies container images and execution types

#### Webhook CRDs

- **`Webhook`** (`executor.testkube.io/v1`)
  - **Definition**: [`api/executor/v1/webhook_types.go`](api/executor/v1/webhook_types.go)
  - **Purpose**: Defines webhooks triggered by test execution events

- **`WebhookTemplate`** (`executor.testkube.io/v1`)
  - **Definition**: [`api/executor/v1/webhook_types.go`](api/executor/v1/webhook_types.go)
  - **Purpose**: Reusable webhook templates with configurable payloads

#### Other CRDs

- **`TestTrigger`** (`tests.testkube.io/v1`)
  - **Definition**: [`api/testtriggers/v1/testtrigger_types.go`](api/testtriggers/v1/testtrigger_types.go)
  - **Purpose**: Automatically triggers tests/workflows based on Kubernetes events
  - **Features**: Watches Pods, Deployments, Services, etc. and triggers executions

#### Other Legacy CRDs

- **`Template`** (`tests.testkube.io/v1`)
  - **Definition**: [`api/template/v1/template_types.go`](api/template/v1/template_types.go)
  - **Purpose**: Reusable test templates for common test patterns

- **`Script`** (`tests.testkube.io/v1`, v2)
  - **Definition**: [`api/script/v1/`, `api/script/v2/`](api/script/)
  - **Purpose**: Legacy test script definitions (deprecated in favor of Test CRD)

#### CRD Lifecycle

1. **Definition**: CRDs are defined in Go using Kubebuilder annotations (`+kubebuilder:object:root=true`)
2. **Generation**: `controller-gen` generates CRD YAML files in `k8s/crd/`
3. **Post-processing**: CRD files are optimized to reduce size (for Kubernetes annotation limits)
4. **Deployment**: CRDs are installed via Helm chart (`k8s/helm/testkube/crds/`)
5. **API Server**: Kubernetes API server validates and stores CRD instances
6. **Controllers**: Controllers watch CRDs and take actions (see [Kubernetes Controllers](#2-kubernetes-controllers))

## Data Flow: TestWorkflow Execution

1. **Creation**: User creates a `TestWorkflowExecution` CRD (via CLI or YAML)
2. **Controller Watch**: TestWorkflowExecution controller detects the CRD
3. **Scheduling**: Controller calls the executor (`pkg/testworkflows/testworkflowexecutor/`)
4. **Pod Creation**: Executor creates Kubernetes Pods with:
   - Init container: `testworkflow-init` (orchestrates workflow steps)
   - Step containers: Execute individual workflow steps
   - Toolkit container: `testworkflow-toolkit` (collects artifacts)
5. **Execution**: Pods run workflow steps, collect outputs, upload artifacts to MinIO
6. **Status Updates**: Execution status is written to PostgreSQL and CRD status
7. **Events**: Lifecycle events published to NATS and processed by event listeners (webhooks, etc.)

You can read more about how Workflows are executed in our docs at https://docs.testkube.io/articles/test-workflows-high-level-architecture. 

## Kubernetes Deployment


**Helm Chart**: [`k8s/helm/testkube/`](k8s/helm/testkube/)

The Helm chart deploys:
- API server deployment
- PostgreSQL (via subchart)
- MinIO (via subchart)
- NATS (via subchart)
- Kubernetes RBAC and service accounts

**Configuration**: See [`DEVELOPMENT.md`](DEVELOPMENT.md) for local development setup using Tilt.

## Key Configuration

**Environment Variables**: [`internal/config/config.go`](internal/config/config.go)
- `ENABLE_K8S_CONTROLLERS` - Enable Kubernetes controllers
- Database, storage, and NATS connection strings
- Feature flags for various agent personas

## CLI

**Entry Point**: [`cmd/kubectl-testkube/main.go`](cmd/kubectl-testkube/main.go)

The Testkube CLI (`kubectl-testkube`, typically invoked as `testkube`) is a kubectl plugin that provides a command-line interface for managing tests, workflows, and executions.

### Architecture

**Command Structure**: [`cmd/kubectl-testkube/commands/`](cmd/kubectl-testkube/commands/)
- Root command and command groups (testworkflows, webhooks, artifacts, etc.)
- Common utilities: [`cmd/kubectl-testkube/commands/common/`](cmd/kubectl-testkube/commands/common/)
- Client abstraction: Works with both standalone API and control plane APIs

**Client Layer**:
- [`pkg/newclients/`](pkg/newclients/) - API clients for tests, testworkflows, webhooks
- [`pkg/controlplaneclient/`](pkg/controlplaneclient/) - Control plane client
- [`cmd/kubectl-testkube/config/`](cmd/kubectl-testkube/config/) - Configuration management (API server URIs, contexts)

**Configuration**: CLI stores configuration in `~/.testkube/` directory, including:
- API server endpoints (standalone or control plane)
- Authentication tokens
- Contexts (for multi-environment setups)

## Related Documentation

- [`DEVELOPMENT.md`](DEVELOPMENT.md) - Local development guide
- [`CONTRIBUTING.md`](CONTRIBUTING.md) - Contribution guidelines
