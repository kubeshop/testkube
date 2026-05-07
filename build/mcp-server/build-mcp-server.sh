#!/bin/bash

# Build script for Testkube MCP Server Docker image
# Usage: ./scripts/build-mcp-server.sh [tag]

set -e

# Default values
TAG=${1:-"kubeshop/mcp-server:latest"}
VERSION=${VERSION:-"1.0.0"}
GIT_SHA=${GIT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}

echo "Building Testkube MCP Server Docker image..."
echo "Tag: $TAG"
echo "Version: $VERSION"
echo "Git SHA: $GIT_SHA"

# Build using docker buildx for multi-platform support
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --file build/mcp-server/Dockerfile \
  --build-arg VERSION="$VERSION" \
  --build-arg GIT_SHA="$GIT_SHA" \
  --label "org.opencontainers.image.version=$VERSION" \
  --label "org.opencontainers.image.revision=$GIT_SHA" \
  --tag "$TAG" \
  --push \
  .

echo "Successfully built and pushed $TAG"

# Also build locally for testing
LOCAL_TAG="${TAG}-local"
echo "Building local image: $LOCAL_TAG"

# Detect local platform - check both uname and arch
LOCAL_PLATFORM=$(uname -m)
ARCH_PLATFORM=$(arch)

if [[ "$ARCH_PLATFORM" == "arm64" ]] || [[ "$LOCAL_PLATFORM" == "arm64" ]]; then
    PLATFORM="linux/arm64"
elif [[ "$LOCAL_PLATFORM" == "x86_64" ]] || [[ "$ARCH_PLATFORM" == "x86_64" ]]; then
    PLATFORM="linux/amd64"
else
    echo "Unknown platform: uname=$LOCAL_PLATFORM, arch=$ARCH_PLATFORM"
    echo "Defaulting to linux/amd64"
    PLATFORM="linux/amd64"
fi

echo "Building for local platform: $PLATFORM"

docker buildx build \
  --platform "$PLATFORM" \
  --file build/mcp-server/Dockerfile \
  --build-arg VERSION="$VERSION" \
  --build-arg GIT_SHA="$GIT_SHA" \
  --label "org.opencontainers.image.version=$VERSION" \
  --label "org.opencontainers.image.revision=$GIT_SHA" \
  --tag "$LOCAL_TAG" \
  --load \
  .

echo "Local image built: $LOCAL_TAG"
echo ""
echo "To test the MCP server locally:"
echo "docker run --rm -it -e TK_ACCESS_TOKEN=your_token -e TK_ORG_ID=your_org -e TK_ENV_ID=your_env $LOCAL_TAG"
