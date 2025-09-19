# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE=alpine:3.20.6

# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=1.0.0
ARG GIT_SHA=unknown

WORKDIR /build
COPY . .

# Build the kubectl-testkube binary with MCP support
RUN cd cmd/kubectl-testkube && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build \
    -ldflags "-X github.com/kubeshop/testkube/pkg/version.Version=${VERSION} \
              -X github.com/kubeshop/testkube/pkg/version.Commit=${GIT_SHA}" \
    -o /app/kubectl-testkube \
    -mod mod -a .

# Final stage
FROM ${ALPINE_IMAGE}
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh -u 1001 mcpuser

# Copy the binary
COPY --from=build /app/kubectl-testkube /bin/kubectl-testkube
RUN chmod +x /bin/kubectl-testkube

# Create symbolic links for convenience
RUN ln -s /bin/kubectl-testkube /bin/testkube
RUN ln -s /bin/kubectl-testkube /bin/tk

# Create config directory
RUN mkdir -p /.testkube && \
    echo "{}" > /.testkube/config.json && \
    chown -R mcpuser:mcpuser /.testkube && \
    chmod 755 /.testkube && \
    chmod 660 /.testkube/config.json

# Set working directory
WORKDIR /home/mcpuser

# Switch to non-root user
USER mcpuser

# Set environment variables for MCP server
ENV TK_CONTROL_PLANE_URL=https://api.testkube.io
ENV TK_DEBUG=false
ENV TK_MCP_ENV_MODE=true

# Expose port not needed for now, this MCP server uses stdio
# EXPOSE 8080

# Default command runs the MCP server
ENTRYPOINT ["/bin/kubectl-testkube", "mcp", "serve"]
