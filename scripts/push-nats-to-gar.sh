#!/usr/bin/env bash
# Copy the upstream NATS images used by the testkube chart to Google Artifact
# Registry, so the chart can be installed from GAR only.
#
# Image names and tags are read from the vendored NATS subchart. Each image is
# published as <gar-repository>/<image basename>:<tag>, e.g. nats:2.15.0-alpine.
# Existing GAR tags are kept (see KEEP_EXISTING_TAGS in push-to-gar.sh), since
# upstream images are rebuilt under the same tag.
#
# Usage: ./scripts/push-nats-to-gar.sh <gar-repository> [nats-values-file]
# Requires yq, docker buildx and being logged in to both registries.

set -euo pipefail

GAR_REPOSITORY="${1:-}"
VALUES_FILE="${2:-k8s/helm/testkube/charts/nats/values.yaml}"

if [ -z "$GAR_REPOSITORY" ]; then
  echo "Usage: $0 <gar-repository> [nats-values-file]"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

failed=0
for component in container reloader; do
  registry="$(yq ".${component}.image.registry // \"docker.io\"" "$VALUES_FILE")"
  repository="$(yq ".${component}.image.repository" "$VALUES_FILE")"
  tag="$(yq ".${component}.image.tag" "$VALUES_FILE")"
  if [ "$repository" = "null" ] || [ "$tag" = "null" ]; then
    echo "Error: ${component}.image.repository/tag not found in ${VALUES_FILE}"
    failed=1
    continue
  fi
  if [ "$registry" = "docker.io" ] && [[ "$repository" != */* ]]; then
    repository="library/${repository}"
  fi

  KEEP_EXISTING_TAGS=true "${SCRIPT_DIR}/push-to-gar.sh" "$tag" "$GAR_REPOSITORY" \
    "${registry}/${repository}=${repository##*/}" || failed=1
done

exit "$failed"
