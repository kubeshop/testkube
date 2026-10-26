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
- `TK_MCP_SHTTP_AUTH_TOKEN` - Separate inbound bearer token (required only with `--transport=shttp`)

Use `--debug` for debug output and CLI flags for transport, bind host, and port.

## MCP Client Usage

Use with MCP Inspector for interactive testing:

```bash
npx @modelcontextprotocol/inspector docker run --rm -i \
  -e TK_ACCESS_TOKEN=$TK_ACCESS_TOKEN \
  -e TK_ORG_ID=$TK_ORG_ID \
  -e TK_ENV_ID=$TK_ENV_ID \
  testkube/mcp-server:local
```

## Authenticated HTTP

The operator deploying this container must create and securely store a random inbound
secret (at least 32 bytes of entropy) independently of `TK_ACCESS_TOKEN`, and distribute
it to authorized MCP clients. The server does not provision the token. Inject the stored
value through your secret manager or environment and reuse it on container restarts.
For local HTTP testing, the container needs an explicit external bind, while the
published port is restricted to host loopback:

```bash
docker run --rm \
  -p 127.0.0.1:8080:8080 \
  -e TK_ACCESS_TOKEN -e TK_ORG_ID -e TK_ENV_ID -e TK_MCP_SHTTP_AUTH_TOKEN \
  testkube/mcp-server:local --transport=shttp --shttp-host=0.0.0.0
```

The image entrypoint already includes `mcp serve`; append flags only. The default
transport remains stdio. Setting `TK_MCP_TRANSPORT`, `TK_MCP_SHTTP_HOST`, or
`TK_MCP_SHTTP_PORT` does not configure the CLI.

Clients send `Authorization: Bearer <inbound-token>` on every request. Remote clients
must use HTTPS: configure the built-in `--shttp-tls`, `--shttp-cert-file`, and
`--shttp-key-file` flags with mounted certificates, or terminate TLS at a proxy and
prevent direct access to the HTTP listener. A loopback proxy must rewrite upstream
`Host` to `localhost`. See the [MCP security and rotation guidance](../../pkg/mcp/README.md#streamable-http-shttp-transport).
