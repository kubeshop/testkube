# Testkube Docker MCP Server Build

This directory contains the Docker build configuration and scripts for the Testkube MCP (Model Context Protocol) server.

## Files

- **`Dockerfile`** - Multi-platform Docker image definition
- **`build-mcp-server.sh`** - Build and push script for production
- **`build-local-mcp.sh`** - Local build script for testing

## Quick Start

### Build for Local Testing

```bash
# Build for your local platform (ARM64/AMD64)
./build-local-mcp.sh testkube/mcp-server:local

# Test the container
docker run --rm -it \
  -e TK_ACCESS_TOKEN=your_token \
  -e TK_ORG_ID=your_org_id \
  -e TK_ENV_ID=your_env_id \
  testkube/mcp-server:local
```

### Build for Production

```bash
# Build and push multi-platform image
./build-mcp-server.sh testkube/mcp-server:latest
```

### Using Docker Bake

```bash
# Build using docker-bake.hcl
docker buildx bake mcp-server
```

## Environment Variables

The MCP server supports environment variable mode when `TK_MCP_ENV_MODE=true` (set by default in the Docker image):

- `TK_ACCESS_TOKEN` - Testkube API access token (required)
- `TK_ORG_ID` - Testkube organization ID (required)
- `TK_ENV_ID` - Testkube environment ID (required)
- `TK_CONTROL_PLANE_URL` - Testkube API URL (optional, default: https://api.testkube.io)
- `TK_DASHBOARD_URL` - Testkube dashboard URL (optional, auto-derived)
- `TK_DEBUG` - Enable debug output (optional, default: false)

## MCP Client Usage

Use with MCP Inspector for interactive testing:

```bash
npx @modelcontextprotocol/inspector docker run --rm -i \
  -e TK_ACCESS_TOKEN=$TK_ACCESS_TOKEN \
  -e TK_ORG_ID=$TK_ORG_ID \
  -e TK_ENV_ID=$TK_ENV_ID \
  testkube/mcp-server:local mcp serve
```
