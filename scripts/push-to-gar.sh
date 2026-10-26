#!/usr/bin/env bash
# Copy released images from Docker Hub to Google Artifact Registry.
#
# The copy is digest-preserving (no rebuild), so GAR and Docker Hub serve the
# exact same multi-arch image.
#
# An existing GAR tag is never overwritten: retagging would leave the previous
# digest untagged, and the repository's delete-untagged cleanup policy would
# remove it. Images whose source tag does not exist are skipped with a warning,
# since not every image ships in every release line.
#
# Usage: ./scripts/push-to-gar.sh <version> <gar-repository> <source-image>=<gar-name>...
# Example: ./scripts/push-to-gar.sh 2.14.1 us-east1-docker.pkg.dev/my-project/testkube \
#            docker.io/kubeshop/testkube-api-server=api-server
# Requires being logged in to both registries (docker login).

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

# Prints the manifest digest of a reference, or nothing if the tag does not
# exist. Any other registry error is returned as a failure.
get_digest() {
  local ref="$1" output
  if output="$(docker buildx imagetools inspect "$ref" --format '{{json .Manifest.Digest}}' 2>&1)"; then
    echo "$output" | tail -n 1 | tr -d '"'
    return 0
  fi
  if echo "$output" | grep -q "${ref}: not found"; then
    return 0
  fi
  echo "Error: failed to inspect ${ref}:" >&2
  echo "$output" >&2
  return 1
}

push_image() {
  local source_ref="$1:${VERSION}" gar_ref="${GAR_REPOSITORY}/$2:${VERSION}"
  local source_digest gar_digest pushed_digest

  source_digest="$(get_digest "$source_ref")" || return 1
  if [ -z "$source_digest" ]; then
    echo "::warning::${source_ref} does not exist, skipping"
    return 0
  fi

  gar_digest="$(get_digest "$gar_ref")" || return 1
  if [ "$gar_digest" = "$source_digest" ]; then
    echo "${gar_ref} is already in sync (${source_digest})"
    return 0
  fi
  if [ -n "$gar_digest" ]; then
    echo "::error::${gar_ref} already exists with a different digest (Docker Hub: ${source_digest}, GAR: ${gar_digest}). Refusing to overwrite it."
    return 1
  fi

  echo "Copying ${source_ref} -> ${gar_ref}"
  docker buildx imagetools create --tag "$gar_ref" "$source_ref" || return 1

  pushed_digest="$(get_digest "$gar_ref")" || return 1
  if [ "$pushed_digest" != "$source_digest" ]; then
    echo "::error::digest mismatch for ${gar_ref} (expected ${source_digest}, got ${pushed_digest:-none})"
    return 1
  fi
  echo "Pushed ${gar_ref} (${source_digest})"
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
  push_image "$source_image" "$gar_name" || failed=1
done

exit "$failed"
