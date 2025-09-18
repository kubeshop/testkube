# Docker MCP Catalog Submission Guide

This document outlines how to package and submit the Testkube MCP server to the Docker MCP Catalog.

## Overview

The Testkube MCP server provides comprehensive tools for interacting with Testkube workflows, executions, and artifacts through the Model Context Protocol. It enables AI assistants to manage test automation workflows, monitor test executions, and access test artifacts.

## Files Created

1. **`build/mcp-server/Dockerfile`** - Docker container definition
2. **`mcp-registry-metadata.json`** - Metadata for Docker MCP registry submission
3. **`build/mcp-server/scripts/build-mcp-server.sh`** - Build script for Docker image
4. **`build/mcp-server/scripts/build-local-mcp.sh`** - Local build script for testing
5. **Updated `docker-bake.hcl`** - Added MCP server target

## Building the Docker Image

### Using the Build Script (Recommended)

```bash
# Build and push to registry
./build/mcp-server/scripts/build-mcp-server.sh testkube/mcp-server:latest

# Build locally for testing
./build/mcp-server/scripts/build-local-mcp.sh testkube/mcp-server:local
```

### Using Docker Buildx Directly

```bash
# Build multi-platform image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --file build/mcp-server/Dockerfile \
  --build-arg VERSION="1.0.0" \
  --build-arg GIT_SHA="$(git rev-parse --short HEAD)" \
  --tag testkube/mcp-server:latest \
  --push \
  .
```

### Using Docker Bake

```bash
# Build using docker-bake.hcl
docker buildx bake mcp-server
```

### Using Local Build Script

```bash
# Build for local platform (ARM64/AMD64)
./build/mcp-server/scripts/build-local-mcp.sh testkube/mcp-server:local
```

## Testing the Container

### Prerequisites

Set required environment variables:

```bash
export TK_ACCESS_TOKEN="your_testkube_access_token"
export TK_ORG_ID="your_organization_id"
export TK_ENV_ID="your_environment_id"
```

### Run Tests

```bash
# Run container with MCP server
docker run --rm -it \
  -e TK_ACCESS_TOKEN="$TK_ACCESS_TOKEN" \
  -e TK_ORG_ID="$TK_ORG_ID" \
  -e TK_ENV_ID="$TK_ENV_ID" \
  testkube/mcp-server:local

# Test with MCP Inspector
npx @modelcontextprotocol/inspector docker run --rm -i \
  -e TK_ACCESS_TOKEN="$TK_ACCESS_TOKEN" \
  -e TK_ORG_ID="$TK_ORG_ID" \
  -e TK_ENV_ID="$TK_ENV_ID" \
  testkube/mcp-server:local mcp serve
```

## Docker MCP Registry Submission

### Step 1: Prepare Repository

1. Fork the [Docker MCP Registry](https://github.com/docker/mcp-registry) repository
2. Clone your fork locally

### Step 2: Add MCP Server Metadata

1. Copy `mcp-registry-metadata.json` to the registry repository
2. Place it in the appropriate directory structure (follow the registry's guidelines)
3. Update any required fields specific to the registry format

### Step 3: Submit Pull Request

1. Create a pull request with your MCP server metadata
2. Include a description of the server's capabilities
3. Provide example usage and configuration

### Step 4: Docker Hub Publishing

If using Docker-built images (recommended):

1. Submit metadata to Docker MCP Registry
2. Docker will build, sign, and publish to `mcp/testkube-mcp` namespace
3. Images will be available within 24 hours

If using self-provided images:

1. Push your image to Docker Hub: `docker push testkube/mcp-server:latest`
2. Update metadata to reference your Docker Hub image
3. Submit to registry

## Configuration

### Required Environment Variables

- `TK_ACCESS_TOKEN` - Testkube API access token
- `TK_ORG_ID` - Testkube organization ID  
- `TK_ENV_ID` - Testkube environment ID

### Optional Environment Variables

- `TK_CONTROL_PLANE_URL` - Testkube API URL (default: https://api.testkube.io)
- `TK_DASHBOARD_URL` - Testkube dashboard URL (auto-derived from control plane URL)
- `TK_DEBUG` - Enable debug output (default: false)

### Example Docker Compose

```yaml
version: '3.8'
services:
  testkube-mcp:
    image: testkube/mcp-server:latest
    environment:
      - TK_ACCESS_TOKEN=${TK_ACCESS_TOKEN}
      - TK_ORG_ID=${TK_ORG_ID}
      - TK_ENV_ID=${TK_ENV_ID}
      - TK_DEBUG=false
    command: ["mcp", "serve"]
```

## MCP Client Configuration

### VSCode Configuration

```json
{
  "servers": {
    "testkube": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "TK_ACCESS_TOKEN=${TK_ACCESS_TOKEN}",
        "-e", "TK_ORG_ID=${TK_ORG_ID}",
        "-e", "TK_ENV_ID=${TK_ENV_ID}",
        "testkube/mcp-server:latest",
        "mcp", "serve"
      ],
      "type": "stdio"
    }
  }
}
```

### Claude Desktop Configuration

```json
{
  "mcpServers": {
    "testkube": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "TK_ACCESS_TOKEN=${TK_ACCESS_TOKEN}",
        "-e", "TK_ORG_ID=${TK_ORG_ID}",
        "-e", "TK_ENV_ID=${TK_ENV_ID}",
        "testkube/mcp-server:latest",
        "mcp", "serve"
      ]
    }
  }
}
```

## Available Tools

The MCP server provides 16 tools for comprehensive Testkube management:

### Workflow Management
- `list_workflows` - List workflows with filtering
- `get_workflow` - Get workflow details
- `get_workflow_definition` - Get workflow YAML
- `get_workflow_metrics` - Get workflow performance metrics
- `create_workflow` - Create new workflow
- `update_workflow` - Update existing workflow
- `run_workflow` - Execute workflow

### Execution Management
- `list_executions` - List workflow executions
- `get_execution_info` - Get execution details
- `fetch_execution_logs` - Get execution logs
- `lookup_execution_id` - Resolve execution name to ID
- `abort_workflow_execution` - Cancel running execution

### Artifact Management
- `list_artifacts` - List execution artifacts
- `read_artifact` - Read artifact content

### Utility Tools
- `build_dashboard_url` - Generate dashboard URLs
- `list_labels` - List available labels
- `list_resource_groups` - List resource groups

## Security Considerations

- The container runs as non-root user (UID 1001)
- No sensitive data is stored in the image
- All authentication is handled via environment variables
- Container uses minimal Alpine Linux base image
- Multi-platform builds ensure compatibility

## Troubleshooting

### Common Issues

1. **Authentication Errors**: Verify `TK_ACCESS_TOKEN`, `TK_ORG_ID`, and `TK_ENV_ID` are correct
2. **Network Issues**: Check `TK_CONTROL_PLANE_URL` is accessible
3. **Permission Errors**: Ensure environment variables are properly set

### Debug Mode

Enable debug output for troubleshooting:

```bash
docker run --rm -it \
  -e TK_ACCESS_TOKEN="$TK_ACCESS_TOKEN" \
  -e TK_ORG_ID="$TK_ORG_ID" \
  -e TK_ENV_ID="$TK_ENV_ID" \
  -e TK_DEBUG=true \
  testkube/mcp-server:latest
```

## Next Steps

1. Test the containerized MCP server thoroughly
2. Submit to Docker MCP Registry
3. Monitor for approval and publication
4. Update documentation with Docker Hub links
5. Consider adding to CI/CD pipeline for automated builds
