#!/bin/sh
set -eu

# Testkube — Local Kubernetes Cluster Setup for Tilt Development
#
# Creates a k3d cluster with a local container registry for use with `tilt up`.
#
# Usage:
#   ./scripts/tilt-cluster.sh          # Create cluster + registry
#   ./scripts/tilt-cluster.sh --delete # Delete the cluster + registry

CLUSTER_NAME="testkube-dev"
REGISTRY_NAME="testkube-registry"
REGISTRY_PORT="5001"
ACTION="create"

while [ $# -gt 0 ]; do
    case $1 in
        --delete)
            ACTION=delete
            ;;
        -h|--help)
            echo "Usage: $0 [--delete]"
            echo ""
            echo "Options:"
            echo "  --delete   Delete the cluster and registry"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Run with --help for usage"
            exit 1
            ;;
    esac
    shift
done

# --- Delete ---

if [ "$ACTION" = "delete" ]; then
    echo "Deleting k3d cluster '$CLUSTER_NAME'..."
    k3d cluster delete "$CLUSTER_NAME" 2>/dev/null || true
    echo "Deleting registry '$REGISTRY_NAME'..."
    k3d registry delete "$REGISTRY_NAME" 2>/dev/null || true
    echo "Done."
    exit 0
fi

# --- Create ---

if k3d cluster list 2>/dev/null | grep -q "^${CLUSTER_NAME}"; then
    echo "k3d cluster '$CLUSTER_NAME' already exists."
    echo ""
    echo "  Start it:          k3d cluster start $CLUSTER_NAME"
    echo "  Start developing:  tilt up"
    echo "  Delete cluster:    $0 --delete"
    exit 0
fi

echo "Creating k3d cluster '$CLUSTER_NAME' with local registry..."
k3d cluster create "$CLUSTER_NAME" \
    --registry-create "${REGISTRY_NAME}:0.0.0.0:${REGISTRY_PORT}" \
    --wait

echo ""
echo "k3d cluster '$CLUSTER_NAME' created."
echo "  Context:  k3d-$CLUSTER_NAME"
echo "  Registry: localhost:${REGISTRY_PORT}"
echo ""
echo "Next steps:"
echo "  tilt up                       # Start development"
echo "  tilt up -- --debug            # Start with Delve debugger"
echo "  tilt up -- --db=mongo         # Use MongoDB instead of PostgreSQL"
