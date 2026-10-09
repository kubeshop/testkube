#!/usr/bin/env bash
# Sign released images in Docker Hub and Google Artifact Registry.
#
# Both registries serve the same digest (see push-to-gar.sh), which is resolved
# from Docker Hub and must already exist in GAR. Images whose source tag does
# not exist are skipped with a warning, like push-to-gar.sh does.
#
# Usage: ./scripts/sign-images.sh <version> <gar-repository> <source-image>=<gar-name>...
# Requires cosign, docker buildx and being logged in to both registries.

set -euo pipefail

if [ "$#" -lt 3 ]; then
  echo "Usage: $0 <version> <gar-repository> <source-image>=<gar-name>..."
  exit 1
fi

VERSION="$1"
GAR_REPOSITORY="$2"
shift 2

if ! [[ "$VERSION" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
  echo "Error: invalid version '${VERSION}'"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

sign_image() {
  local source_image="$1" gar_image="${GAR_REPOSITORY}/$2" output digest gar_digest

  if ! output="$(docker buildx imagetools inspect "${source_image}:${VERSION}" --format '{{json .Manifest.Digest}}' 2>&1)"; then
    if echo "$output" | grep -q "${source_image}:${VERSION}: not found"; then
      echo "::warning::${source_image}:${VERSION} does not exist, skipping"
      return 0
    fi
    echo "Error: failed to inspect ${source_image}:${VERSION}:"
    echo "$output"
    return 1
  fi
  digest="$(echo "$output" | tail -n 1 | tr -d '"')"

  gar_digest="$(docker buildx imagetools inspect "${gar_image}:${VERSION}" --format '{{json .Manifest.Digest}}' | tail -n 1 | tr -d '"')" || return 1
  if [ "$gar_digest" != "$digest" ]; then
    echo "::error::${gar_image}:${VERSION} has digest ${gar_digest}, expected ${digest} (Docker Hub)"
    return 1
  fi

  "${SCRIPT_DIR}/cosign-sign.sh" "$source_image" "$digest" || return 1
  "${SCRIPT_DIR}/cosign-sign.sh" "$gar_image" "$digest" || return 1
}

failed=0
for mapping in "$@"; do
  source_image="${mapping%%=*}"
  gar_name="${mapping#*=}"
  if [ -z "$source_image" ] || [ -z "$gar_name" ] || [ "$source_image" = "$mapping" ]; then
    echo "Error: invalid mapping '${mapping}', expected <source-image>=<gar-name>"
    failed=1
    continue
  fi
  sign_image "$source_image" "$gar_name" || failed=1
done

exit "$failed"
