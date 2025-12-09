#!/bin/bash

# Quick build script for local ARM64/AMD64 platform
# Usage: ./scripts/build-local-mcp.sh [tag]

set -e

TAG=${1:-"kubeshop/mcp-server:local"}
VERSION=${VERSION:-"1.0.0"}
GIT_SHA=${GIT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}

# Detect local platform - check both uname and arch
LOCAL_PLATFORM=$(uname -m)
ARCH_PLATFORM=$(arch)

if [[ "$ARCH_PLATFORM" == "arm64" ]] || [[ "$LOCAL_PLATFORM" == "arm64" ]]; then
    PLATFORM="linux/arm64"
    echo "Detected Apple Silicon (ARM64) - building for linux/arm64"
elif [[ "$LOCAL_PLATFORM" == "x86_64" ]] || [[ "$ARCH_PLATFORM" == "x86_64" ]]; then
    PLATFORM="linux/amd64"
    echo "Detected Intel/AMD64 - building for linux/amd64"
else
    echo "Unknown platform: uname=$LOCAL_PLATFORM, arch=$ARCH_PLATFORM"
    echo "Defaulting to linux/amd64"
    PLATFORM="linux/amd64"
fi

echo "Building Testkube MCP Server for local platform..."
echo "Tag: $TAG"
echo "Platform: $PLATFORM"
echo "Version: $VERSION"
echo "Git SHA: $GIT_SHA"
echo ""

# Build for local platform only
docker buildx build \
  --platform "$PLATFORM" \
  --file build/mcp-server/Dockerfile \
  --build-arg VERSION="$VERSION" \
  --build-arg GIT_SHA="$GIT_SHA" \
  --label "org.opencontainers.image.version=$VERSION" \
  --label "org.opencontainers.image.revision=$GIT_SHA" \
  --tag "$TAG" \
  --load \
  .

echo ""
echo "✅ Successfully built $TAG for $PLATFORM"
echo ""
echo "To test the MCP server:"
echo "  export TK_ACCESS_TOKEN=your_token"
echo "  export TK_ORG_ID=your_org_id"
echo "  export TK_ENV_ID=your_env_id"
echo "  ./scripts/test-mcp-server.sh $TAG"
echo ""
echo "Or run directly:"
echo "  docker run --rm -it -e TK_ACCESS_TOKEN=\$TK_ACCESS_TOKEN -e TK_ORG_ID=\$TK_ORG_ID -e TK_ENV_ID=\$TK_ENV_ID $TAG"
