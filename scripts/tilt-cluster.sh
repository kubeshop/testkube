#!/bin/sh
set -euo pipefail

# Testkube — Local Kubernetes Cluster Setup for Tilt Development
#
# Creates a local Kubernetes cluster for use with `tilt up`.
#
# Usage:
#   ./scripts/tilt-cluster.sh          # Default: kind cluster
#   ./scripts/tilt-cluster.sh --kind   # Explicit kind
#   ./scripts/tilt-cluster.sh --k3d    # Use k3d instead
#   ./scripts/tilt-cluster.sh --delete # Delete the cluster

CLUSTER_NAME="testkube-dev"
ACTION="create"
K8S_DISTRO="kind"

while [ $# -gt 0 ]; do
    case $1 in
        --kind)
            K8S_DISTRO=kind
            ;;
        --k3d)
            K8S_DISTRO=k3d
            ;;
        --delete)
            ACTION=delete
            ;;
        -h|--help)
            echo "Usage: $0 [--kind|--k3d] [--delete]"
            echo ""
            echo "Options:"
            echo "  --kind     Use kind (default)"
            echo "  --k3d      Use k3d"
            echo "  --delete   Delete the cluster instead of creating it"
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
    echo "Deleting $K8S_DISTRO cluster '$CLUSTER_NAME'..."
    case $K8S_DISTRO in
        kind)    kind delete cluster --name "$CLUSTER_NAME" ;;
        k3d)     k3d cluster delete "$CLUSTER_NAME" ;;
    esac
    echo "Done."
    exit 0
fi

# --- Create ---

case $K8S_DISTRO in
    kind)
        if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
            echo "kind cluster '$CLUSTER_NAME' already exists."
            echo ""
            echo "  Start developing:  tilt up"
            echo "  Delete cluster:    $0 --delete"
            exit 0
        fi

        echo "Creating kind cluster '$CLUSTER_NAME'..."
        kind create cluster --name "$CLUSTER_NAME" --wait 60s
        echo ""
        echo "kind cluster '$CLUSTER_NAME' created."
        echo "  Context: kind-$CLUSTER_NAME"
        ;;

    k3d)
        if k3d cluster list 2>/dev/null | grep -q "^${CLUSTER_NAME}"; then
            echo "k3d cluster '$CLUSTER_NAME' already exists."
            echo ""
            echo "  Start it:          k3d cluster start $CLUSTER_NAME"
            echo "  Start developing:  tilt up"
            echo "  Delete cluster:    $0 --delete"
            exit 0
        fi

        echo "Creating k3d cluster '$CLUSTER_NAME'..."
        k3d cluster create "$CLUSTER_NAME" \
            --api-port 6550 \
            -p "8088:8088@loadbalancer" \
            --agents 1 \
            --wait
        echo ""
        echo "k3d cluster '$CLUSTER_NAME' created."
        echo "  Context: k3d-$CLUSTER_NAME"
        ;;
esac

echo ""
echo "Next steps:"
echo "  tilt up                       # Start development"
echo "  tilt up -- --debug            # Start with Delve debugger"
echo "  tilt up -- --db=mongo         # Use MongoDB instead of PostgreSQL"
