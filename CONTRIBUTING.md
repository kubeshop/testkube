# Contributing to Testkube

Thanks for your interest in contributing to Testkube! 🎉

If you're new to open-source contributions, there's a great guide on how to start: 
https://github.com/firstcontributions/first-contributions

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Help](#getting-help)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Building Locally](#building-locally)
- [Running Tests](#running-tests)
- [Code Generation](#code-generation)
- [Linting and Code Quality](#linting-and-code-quality)
- [Development Workflow](#development-workflow)
- [Making Contributions](#making-contributions)
- [Code Style Guidelines](#code-style-guidelines)

## Code of Conduct

This project and everyone participating in it is governed by the Testkube [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Help

### Questions and Ideas

- **Questions**: Use our [Slack Workspace](https://testkubeworkspace.slack.com) for ideas, questions and clarifications
- **Issues**: Report bugs or request features via [GitHub Issues](https://github.com/kubeshop/testkube/issues)

## Testkube Overview

Check out the [Testkube Overview](https://docs.testkube.io/articles/open-source) to understand what components the Open-Source Agent of Testkube has and what they do. 

## Project Structure

Testkube follows a standard Go project layout with clear separation of concerns:

```
testkube/
├── cmd/                   # Application entry points
│   ├── api-server/        # Main API server (Control Plane)
│   ├── kubectl-testkube/  # CLI tool (kubectl plugin)
│   ├── testworkflow-toolkit/  # TestWorkflow execution toolkit
│   └── testworkflow-init/     # TestWorkflow initialization
│   └── ...                # Other cmd-related packages
├── pkg/                   # Public library code (importable by external apps)
│   ├── api/               # API models and clients
│   ├── repository/        # Data repository interfaces
│   ├── testworkflows/     # TestWorkflow processing logic
│   ├── controlplane/      # Control Plane communication
│   └── ...                # Other public packages
├── internal/              # Private application code (not importable)
│   ├── app/               # Application-specific logic
│   ├── config/            # Configuration management
│   └── sync/              # Synchronization logic
│   └── ...                # Other internal packages
├── api/                   # API definitions and OpenAPI specs
├── k8s/                   # Kubernetes manifests and Helm charts
├── proto/                 # Protocol Buffer definitions
├── test/                  # Test fixtures and integration tests
└── build/                 # Build scripts and Dockerfiles
```

### Key Directories Explained

- **`cmd/`**: Contains the main entry points for each binary. Each subdirectory is a separate application.
- **`pkg/`**: Public library code that can be imported by external applications. This is the public API of Testkube.
- **`internal/`**: Private application code that cannot be imported by external packages (enforced by Go compiler).
- **`api/`**: OpenAPI specifications and API contract definitions.
- **`k8s/`**: Kubernetes Custom Resource Definitions (CRDs) and Helm charts for deployment.

## Prerequisites

Before you start developing, ensure you have:

- **Go 1.25+**: Testkube requires Go 1.25 or later. Check with `go version`
- **Make**: Used for build automation (most targets are in the Makefile)
- **Docker**: Required for running integration tests and building container images
- **Kubernetes cluster** (optional): For testing Kubernetes integrations locally
- **kubectl** (optional): For interacting with Kubernetes clusters

### Optional Tools

- **swagger-codegen**: For OpenAPI code generation (install manually)
- **golangci-lint**: Automatically installed via Makefile, but can be installed globally
- **sqlc**: Automatically installed via Makefile for SQL code generation

## Building Locally

Testkube uses a comprehensive Makefile for all build operations. Start by exploring available targets:

```bash
make help
```

### Quick Start

Build all binaries:

```bash
make build
```

This builds:
- `bin/app/api-server` - API server binary
- `bin/app/testkube` - CLI binary
- `bin/app/testworkflow-toolkit` - TestWorkflow toolkit
- `bin/app/testworkflow-init` - TestWorkflow init binary
- `bin/app/kubectl-testkube`- Testkube kubectl plugin 

### Building Individual Components

```bash
# Build API server
make build-api-server

# Build CLI
make build-testkube-cli

# Build TestWorkflow toolkit
make build-toolkit

# Build TestWorkflow init
make build-init
```

### Running the API Server Locally

For local development, you'll need MongoDB and NATS running:

```bash
# Start MongoDB and NATS in Docker
make run-mongo
make run-nats

# Run the API server
make run-api
```

Or start everything at once:

```bash
make dev
```

Stop services when done:

```bash
make stop-mongo
make stop-nats
```

### Cross-Platform Builds

Build for specific platforms:

```bash
# Build for Linux AMD64
make build GOOS=linux GOARCH=amd64

# Build for all supported platforms
make build-all-platforms
```

## Running Tests

### Unit Tests

Run all unit tests:

```bash
make unit-tests
```

This runs tests with coverage reporting. View coverage:

```bash
make cover
```

### Integration Tests

Integration tests require additional setup (MongoDB, MinIO, etc.):

```bash
make integration-tests
```

Integration tests are identified by the `_Integration` suffix in test names.

### Running Specific Tests

Use standard Go test commands:

```bash
# Run tests in a specific package
go test ./pkg/repository/...

# Run a specific test
go test -run TestSpecificFunction ./pkg/repository

# Run tests with verbose output
go test -v ./pkg/...
```

### Test Coverage

We target 80% code coverage. Check coverage:

```bash
# Generate coverage report
make unit-tests
go tool cover -html=coverage.out
```

## Code Generation

Testkube uses several code generation tools. Always regenerate code after modifying:

- **Protobuf definitions** (`proto/`)
- **OpenAPI schemas** (`api/`)

### Generate All Code

```bash
make generate
```

This runs all generation targets:
- `generate-protobuf` - Generates Go code from `.proto` files
- `generate-openapi` - Generates API models from OpenAPI specs
- `generate-mocks` - Generates mock interfaces using mockgen
- `generate-sqlc` - Generates database query code
- `generate-crds` - Generates Kubernetes CRDs

### Individual Generation Targets

```bash
# Generate protobuf code
make generate-protobuf

# Generate OpenAPI models (requires swagger-codegen)
make generate-openapi

# Generate mocks
make generate-mocks

# Generate SQLC queries
make generate-sqlc

# Generate Kubernetes CRDs
make generate-crds
```

**Important**: Always run `make generate` before committing if you've modified:
- `.proto` files
- OpenAPI specifications
- SQL queries in `pkg/database/`
- Interface definitions that need mocks

## Linting and Code Quality

### Running Linters

```bash
# Run all linters
make lint

# Run linters with automatic fixes
make lint-fix
```

The project uses `golangci-lint` with the following enabled linters:
- `govet` - Go vet checks
- `revive` - Go linter with additional rules
- `staticcheck` - Static analysis
- `unused` - Find unused code
- `ineffassign` - Detect ineffectual assignments

### Code Formatting

Code is automatically formatted using `goimports` (via golangci-lint). The formatter:
- Organizes imports (standard library, third-party, local)
- Formats code according to `gofmt` rules

Run formatting:

```bash
make lint-fix
```

Or manually:

```bash
goimports -w .
```

## Development Workflow

### 1. Fork and Clone

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/testkube.git
cd testkube
git remote add upstream https://github.com/kubeshop/testkube.git
```

### 2. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

Use descriptive branch names:
- `feature/` - New features
- `fix/` - Bug fixes
- `refactor/` - Code refactoring

### 3. Make Changes

- Write your code following the [Code Style Guidelines](#code-style-guidelines)
- Add tests for new functionality
- Update documentation if needed
- Ensure all tests pass: `make test`
- Run linters: `make lint`

### 4. Generate Code (if needed)

If you modified protobuf, OpenAPI, SQL, or interfaces:

```bash
make generate
```

### 5. Commit Changes

Write clear, descriptive commit messages:

```
feat: add support for custom test executors

- Add ExecutorConfig interface
- Implement executor registry
- Add tests for executor registration

Fixes #123
```

### 6. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub.

## Making Contributions

### How to Contribute

1. **Fix Issues**: Pick an issue from [GitHub Issues](https://github.com/kubeshop/testkube/issues) and submit a PR
2. **Implement Features**: Discuss features in [Ideas discussions](https://github.com/kubeshop/testkube/discussions/categories/ideas) first
3. **Improve Documentation**: Documentation for Testkube is in the [kubeshop/testkube-docs](https://github.com/kubeshop/testkube-docs) repository, open a corresponding PR there for documentation updates.
4. **Report Bugs**: Use GitHub Issues to report problems

### Pull Request Process

1. **Update your branch**: Before submitting, rebase on latest `main`:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Ensure quality**:
   - All tests pass: `make test`
   - Code is formatted: `make lint-fix`
   - Linters pass: `make lint`
   - Code generation is up to date: `make generate`

3. **Write a good PR description**:
   - What changes were made
   - Why the changes were needed
   - How to test the changes
   - Related issues/PRs

4. **Request review**: Assign reviewers or mention maintainers

### PR Checklist

Before submitting, ensure:

- [ ] Code follows style guidelines
- [ ] Tests added/updated and passing
- [ ] Documentation updated (if needed)
- [ ] Code generation run (`make generate`)
- [ ] Linters pass (`make lint`)
- [ ] Commit messages are clear and descriptive
- [ ] Branch is up to date with `main`

## Code Style Guidelines

### Go Code Style

- **Formatting**: Always use `gofmt` (enforced via `goimports`)
- **Naming**: Follow Go naming conventions:
  - Exported names start with capital letters
  - Use clear, descriptive names
  - Avoid abbreviations unless widely understood

- **Error Handling**: Always handle errors explicitly:
  ```go
  if err != nil {
      return fmt.Errorf("context: %w", err)
  }
  ```

- **Testing**: 
  - Write tests for all new functionality
  - Use table-driven tests when appropriate
  - Aim for 80% code coverage
  - Name test files `*_test.go`
  - Use descriptive test names: `TestFunctionName_Scenario_ExpectedResult`

- **Documentation**: 
  - Document all exported functions, types, and packages
  - Use complete sentences in comments
  - Add examples for complex functions

### Project-Specific Guidelines

- **Package Organization**: 
  - `pkg/` for public, importable code
  - `internal/` for private application code
  - Keep packages focused and cohesive

- **Dependencies**: 
  - Minimize external dependencies
  - Prefer standard library when possible
  - Keep dependencies up to date

- **Kubernetes Resources**: 
  - Use Helm charts for deployment (`k8s/helm/`)
  - Comment non-obvious Kubernetes configurations
  - Keep CRD definitions in sync with Go types

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Testkube Documentation](https://docs.testkube.io)
- [Testkube Slack Community](https://testkubeworkspace.slack.com)

## Questions?

If you have questions or need help:

- Join our [Slack workspace](https://testkubeworkspace.slack.com)
- Open an issue for bugs or feature requests

Thank you for contributing to Testkube! 🚀

