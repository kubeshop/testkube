# Contributing to Testkube Open Source

Thanks for reaching out - lovely to have you here!

Testkube consists of two major components:

1. The Testkube Agent (this repo), which is 100% Open Source and can be used to leverage core Testkube functionality for free.
2. The Testkube Control Plane (commercial), which adds a management layer and user-friendly dashboard to the Testkube experience.

Read more about Testkube and its components/functionality in our [Documentation](https://docs.testkube.io).

## Table of Contents

- [Where Can I Help?](#where-can-i-help)
- [Before you Get Started](#before-you-get-started)
- [Coding Prerequisites](#coding-prerequisites)
- [Building the Code](#building-the-code)
- [Running Tests](#running-tests)
- [Linting and Code Style](#linting-and-code-style)
- [Code Generation](#code-generation)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting](#issue-reporting)
- [Using AI to Contribute Code](#using-ai-to-contribute-code)
- [Key Files for New Contributors](#key-files-for-new-contributors)

## Where Can I Help?

If you have a specific idea on how to improve Testkube, please share in our [Slack Channel](https://bit.ly/testkube-slack) or open a corresponding issue here on GitHub. 

If not, there are [many Issues](https://github.com/kubeshop/testkube/issues) to dig into. Simply fork our repo and create a new Pull Request with your code changes. Look for issues labeled `good first issue` or `help wanted` if you're not sure where to start.

> If you're new to the open-source community, there is a nice guide on how to start contributing to projects: https://github.com/firstcontributions/first-contributions

## Before you Get Started

- Make sure you've read our [Code of Conduct](CODE_OF_CONDUCT.md)
- Sign up for our [Slack Channel](https://bit.ly/testkube-slack) where you can ask questions, share ideas, and get help.
- Read the high-level [Open Source Documentation](https://docs.testkube.io/articles/open-source) to make sure you have an understanding of what Testkube Open Source is capable of.
- Read the [Architecture](ARCHITECTURE.md) guide for the Testkube Agent (this repo).
- Read the [Development Guide](DEVELOPMENT.md) to help you set up a local development environment using Tilt.
- Check out the sections below to help you navigate the codebase and understand our development workflow.

## Coding Prerequisites

Before you begin, ensure you have the following installed:

- **Go** 1.25 or later
- **Docker** with BuildX support
- **Kubernetes cluster** (for integration testing) 
- **Helm** v3.x (for local deployment)
- Optional: **[Tilt](https://docs.tilt.dev/install.html)** v0.30.0 or later for local development, see [DEVELOPMENT.md](DEVELOPMENT.md)

To verify your Go installation:

```bash
go version
# Should output: go version go1.25.x ...
```

## Building the Code

All build commands are managed through the Makefile. Run `make help` to see all available targets.

### Build All Binaries

```bash
make build
```

This builds:

- `bin/app/api-server` - The main API server
- `bin/app/testkube` - The Testkube CLI
- `bin/app/testworkflow-init` - Init container for Test Workflow execution
- `bin/app/testworkflow-toolkit` - Runtime utilities for Test Workflow containers

### Build Individual Components

```bash
# Build only the API server
make build-api-server

# Build only the CLI
make build-testkube-cli

# Build the kubectl-testkube plugin
make build-kubectl-testkube-cli

# Build Test Workflow components
make build-toolkit
make build-init
```

### Cross-Platform Builds

```bash
# Build for a specific OS/architecture
make build GOOS=linux GOARCH=amd64

# Build for all supported platforms
make build-all-platforms
```

### Docker Images

```bash
# Build all Docker images
make docker-build

# Build specific images
make docker-build-api
make docker-build-cli
```

## Running Tests

### Unit Tests

```bash
# Run all unit tests with coverage
make test
# or equivalently
make unit-tests
```

### Integration Tests

Integration tests require the Test Workflow components to be built first:

```bash
make integration-tests
```

Integration tests are identified by the `_Integration` suffix in test function names.

### View Coverage Report

```bash
make cover
```

This generates an HTML coverage report and opens it in your browser.

### Running Specific Tests

```bash
# Run tests for a specific package
go test ./pkg/testworkflows/... -v

# Run with race detection
go test ./pkg/... -race

# Run a specific test
go test ./pkg/event/... -run TestEmitter -v
```

## Linting and Code Style

We use [golangci-lint](https://golangci-lint.run/) v2 for static analysis. The configuration is in [`.golangci.yml`](.golangci.yml).

### Run Linter

```bash
make lint
```

This runs the following linters:

- `govet` - Reports suspicious constructs
- `revive` - Extensible Go linter
- `staticcheck` - State-of-the-art static analysis
- `unused` - Checks for unused code
- `ineffassign` - Detects ineffectual assignments

### Auto-fix Issues

```bash
make lint-fix
```

This automatically fixes issues where possible, including formatting with `goimports`.

### Import Ordering

Imports should be organized in the following order (enforced by `goimports`):

1. Standard library imports
2. Third-party imports
3. Local imports (`github.com/kubeshop/testkube`)

Example:

```go
import (
    "context"
    "fmt"

    "github.com/gofiber/fiber/v2"
    "go.uber.org/zap"

    "github.com/kubeshop/testkube/pkg/api/v1/testkube"
    "github.com/kubeshop/testkube/pkg/log"
)
```

### Editor Integration

Most editors support golangci-lint integration:

- **VSCode**: Install the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.Go) and set `"go.lintTool": "golangci-lint"`
- **GoLand**: Enable golangci-lint in Preferences > Tools > Go Linter

## Code Generation

Several parts of the codebase are generated. After making changes to source definitions, regenerate the artifacts:

### Generate All

```bash
make generate
```

### Individual Generation Targets

```bash
# Regenerate OpenAPI models after editing api/v1/testkube.yaml
make generate-openapi

# Regenerate SQL client code after editing database queries
make generate-sqlc

# Regenerate mocks for testing
make generate-mocks

# Regenerate Kubernetes CRDs after editing api/ type definitions
make generate-crds

# Regenerate protobuf code
make generate-protobuf
```

**Important**: Always commit generated files alongside your source changes. Reviewers will verify that generated code is up to date.

## Commit Guidelines

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification for commit messages.

### Commit Message Format

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that don't affect code meaning (formatting, etc.)
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools
- `ci`: Changes to CI configuration files and scripts

### Examples

```
feat(api): add support for workflow templates

fix(cli): correct output format for JSON responses

docs(readme): update installation instructions

refactor(executor): simplify step processing logic
```

### Keep Commits Focused

- Each commit should represent a single logical change
- Split large changes into multiple commits when possible
- Avoid mixing unrelated changes in a single commit

## Pull Request Process

1. **Fork the Repository**: Create your own fork of the repository on GitHub.

2. **Create a Branch**: Create a feature branch from `main`:
   ```bash
   git checkout -b feat/your-feature-name
   ```

3. **Make Your Changes**: Implement your changes following the guidelines in this document.

4. **Verify Your Changes**:
   ```bash
   # Ensure code compiles
   make build
   
   # Run linter
   make lint
   
   # Run tests
   make test
   
   # Regenerate any generated code if needed
   make generate
   ```

5. **Commit Your Changes**: Follow the [commit guidelines](#commit-guidelines).

6. **Push to Your Fork**:
   ```bash
   git push origin feat/your-feature-name
   ```

7. **Open a Pull Request**: 
   - Use a clear, descriptive title
   - Fill out the [PR template](.github/pull_request_template.md)
   - Link any related issues
   - Request review from maintainers

### PR Checklist

Before submitting your PR, ensure:

- [ ] Code compiles without errors (`make build`)
- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Generated code is up to date (`make generate`)
- [ ] Documentation PR is created if needed in the https://github.com/testkube-docs repo)
- [ ] Breaking changes are clearly documented
- [ ] New features include tests

### Review Process

- At least one maintainer approval is required
- CI checks must pass
- Address all review comments before merging
- Maintainers may request changes or ask questions

## Issue Reporting

### Bug Reports

When reporting bugs, please include:

1. **Environment Information**:
   - Testkube version (`testkube version`)
   - Kubernetes version (`kubectl version`)
   - Operating system and architecture

2. **Steps to Reproduce**: Minimal steps to reproduce the issue

3. **Expected Behavior**: What you expected to happen

4. **Actual Behavior**: What actually happened

5. **Logs**: Relevant log output (use code blocks)

6. **Configuration**: Any relevant configuration (sanitize sensitive data)

Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md) when creating issues.

### Feature Requests

For feature requests, please describe:

- The problem you're trying to solve
- Your proposed solution
- Alternative solutions you've considered
- Any additional context

Use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.md) when creating issues.

## Using AI to contribute code

Feel free to use AI to generate new code/features, just make sure to use our [AGENTS.md](AGENTS.md) file
with your IDE/Agents to ensure that generated code is in line with our guidelines and best practices.

## Key Files for New Contributors

This section highlights the most important files to familiarize yourself with when starting to contribute to Testkube.

### Entry Points

| File | Purpose |
|------|---------|
| [`cmd/api-server/main.go`](cmd/api-server/main.go) | Main API server entry point - start here to understand how the agent boots up |
| [`cmd/kubectl-testkube/main.go`](cmd/kubectl-testkube/main.go) | CLI entry point - the `testkube` command users interact with |
| [`cmd/testworkflow-init/main.go`](cmd/testworkflow-init/main.go) | Init container that orchestrates TestWorkflow step execution |
| [`cmd/testworkflow-toolkit/main.go`](cmd/testworkflow-toolkit/main.go) | Runtime utilities available inside TestWorkflow containers |

### Core Type Definitions

| File | Purpose |
|------|---------|
| [`api/testworkflows/v1/testworkflow_types.go`](api/testworkflows/v1/testworkflow_types.go) | TestWorkflow CRD type definition - the main abstraction for running tests |
| [`api/testworkflows/v1/step_types.go`](api/testworkflows/v1/step_types.go) | Step types that define workflow actions (run, shell, artifacts, etc.) |
| [`api/testtriggers/v1/testtrigger_types.go`](api/testtriggers/v1/testtrigger_types.go) | TestTrigger CRD for event-based test execution |
| [`api/executor/v1/webhook_types.go`](api/executor/v1/webhook_types.go) | Webhook and WebhookTemplate CRD definitions |
| [`pkg/api/v1/testkube/`](pkg/api/v1/testkube/) | Generated OpenAPI models used throughout the codebase |

### Configuration

| File | Purpose |
|------|---------|
| [`internal/config/config.go`](internal/config/config.go) | All environment variables that configure the API server (search for `envconfig:"..."` tags) |
| [`api/v1/testkube.yaml`](api/v1/testkube.yaml) | OpenAPI specification defining the REST API contract |
| [`k8s/helm/testkube/values.yaml`](k8s/helm/testkube/values.yaml) | Helm chart default values - authoritative deployment configuration |

### API Layer

| File | Purpose |
|------|---------|
| [`internal/app/api/v1/server.go`](internal/app/api/v1/server.go) | API initialization and route registration (`TestkubeAPI.Init()`) |
| [`internal/app/api/v1/testworkflows.go`](internal/app/api/v1/testworkflows.go) | TestWorkflow CRUD handlers |
| [`internal/app/api/v1/testworkflowexecutions.go`](internal/app/api/v1/testworkflowexecutions.go) | Execution management handlers |
| [`internal/app/api/v1/webhook.go`](internal/app/api/v1/webhook.go) | Webhook management handlers |
| [`pkg/server/httpserver.go`](pkg/server/httpserver.go) | HTTP server setup with Fiber framework |

### Business Logic

| File | Purpose |
|------|---------|
| [`pkg/testworkflows/testworkflowexecutor/`](pkg/testworkflows/testworkflowexecutor/) | Core TestWorkflow execution logic |
| [`pkg/testworkflows/testworkflowprocessor/`](pkg/testworkflows/testworkflowprocessor/) | Workflow processing and step transformation |
| [`pkg/event/emitter.go`](pkg/event/emitter.go) | Event system that publishes execution lifecycle events |
| [`pkg/event/kind/webhook/`](pkg/event/kind/webhook/) | Webhook event listener implementation |
| [`pkg/triggers/`](pkg/triggers/) | TestTrigger service - watches Kubernetes events |
| [`pkg/controller/`](pkg/controller/) | Kubernetes controllers for CRD reconciliation |

### Client Libraries

| File | Purpose |
|------|---------|
| [`pkg/api/v1/client/`](pkg/api/v1/client/) | Low-level API client for direct HTTP calls |
| [`pkg/newclients/`](pkg/newclients/) | Higher-level clients for workflows, webhooks, triggers |
| [`pkg/controlplaneclient/`](pkg/controlplaneclient/) | Client for Control Plane communication (Connected mode) |
| [`cmd/kubectl-testkube/commands/`](cmd/kubectl-testkube/commands/) | CLI command implementations - good examples of client usage |

### Storage & Repositories

| File | Purpose |
|------|---------|
| [`pkg/repository/testworkflow/`](pkg/repository/testworkflow/) | TestWorkflow result/output repository interfaces |
| [`pkg/repository/postgres/`](pkg/repository/postgres/) | PostgreSQL implementation |
| [`pkg/storage/`](pkg/storage/) | Artifact storage interface |
| [`pkg/storage/minio/`](pkg/storage/minio/) | MinIO artifact storage implementation |

### Infrastructure & Logging

| File | Purpose |
|------|---------|
| [`pkg/log/log.go`](pkg/log/log.go) | Logging setup with zap - use `log.DefaultLogger` throughout |
| [`internal/app/api/metrics/metrics.go`](internal/app/api/metrics/metrics.go) | Prometheus metrics definitions |
| [`k8s/helm/testkube/`](k8s/helm/testkube/) | Helm chart for deployment |
| [`k8s/crd/`](k8s/crd/) | Generated CRD YAML files |

### Development & Build

| File | Purpose |
|------|---------|
| [`Makefile`](Makefile) | Build targets - run `make help` for available commands |
| [`Tiltfile`](Tiltfile) | Tilt configuration for local development |
| [`go.mod`](go.mod) | Go module dependencies |
| [`.goreleaser.yml`](.goreleaser.yml) | Release configuration |

### Testing

| File | Purpose |
|------|---------|
| [`test/`](test/) | Integration test fixtures and examples |
| [`internal/test/framework/`](internal/test/framework/) | Test framework utilities |

### Recommended Reading Order

For new contributors, we recommend exploring the codebase in this order:

1. **Start with the types**: Read [`api/testworkflows/v1/testworkflow_types.go`](api/testworkflows/v1/testworkflow_types.go) to understand the core TestWorkflow abstraction
2. **Understand configuration**: Review [`internal/config/config.go`](internal/config/config.go) to see what environment variables drive behavior
3. **Trace the API server startup**: Follow [`cmd/api-server/main.go`](cmd/api-server/main.go) to see how components are wired together
4. **Explore the API layer**: Look at [`internal/app/api/v1/server.go`](internal/app/api/v1/server.go) to understand route registration
5. **Study the CLI**: Browse [`cmd/kubectl-testkube/commands/`](cmd/kubectl-testkube/commands/) for examples of how clients interact with the API
6. **Set up local development**: Follow [`DEVELOPMENT.md`](DEVELOPMENT.md) to run Testkube locally with Tilt

## License

By contributing to Testkube, you agree that your contributions will be licensed under the [MIT License](LICENSE).

## Getting Help

If you need help or have questions:

- **Slack**: Join our [Slack community](https://bit.ly/testkube-slack) for real-time discussions
- **GitHub Issues**: Search existing [issues](https://github.com/kubeshop/testkube/issues) or create a new one
- **Documentation**: Check our [official documentation](https://docs.testkube.io)

## Thank You

Thank you for contributing to Testkube! Your contributions help make test orchestration on Kubernetes better for everyone.
