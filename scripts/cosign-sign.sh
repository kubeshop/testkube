#!/usr/bin/env bash
# Sign <repository>@<digest> with cosign (keyless in GitHub Actions).
#
# Signatures use the classic format (tag sha256-<digest>.sig) instead of the
# cosign v3 default, which stores them as untagged OCI referrers: GAR's
# delete-untagged cleanup policy would remove those. `cosign verify` finds
# either format with default flags. The classic format can't be combined with
# the Sigstore TUF signing config, so the public-good Fulcio/Rekor defaults are
# used instead.
#
# Skips digests that already carry a signature, so re-runs don't stack
# duplicate signatures.
#
# Usage: ./scripts/cosign-sign.sh <repository> <digest>
# Requires cosign, docker buildx and being logged in to the registry.

set -euo pipefail

REPOSITORY="${1:-}"
DIGEST="${2:-}"

if [ -z "$REPOSITORY" ] || [ -z "$DIGEST" ]; then
  echo "Usage: $0 <repository> <digest>"
  exit 1
fi

if ! [[ "$DIGEST" =~ ^sha256:[0-9a-f]{64}$ ]]; then
  echo "Error: invalid digest '${DIGEST}'"
  exit 1
fi

SIGNATURE_REF="${REPOSITORY}:${DIGEST/:/-}.sig"

if output="$(docker buildx imagetools inspect "$SIGNATURE_REF" 2>&1)"; then
  echo "${REPOSITORY}@${DIGEST} is already signed"
  exit 0
fi
if ! echo "$output" | grep -q "${SIGNATURE_REF}: not found"; then
  echo "Error: failed to inspect ${SIGNATURE_REF}:"
  echo "$output"
  exit 1
fi

cosign sign --yes --new-bundle-format=false --use-signing-config=false "${REPOSITORY}@${DIGEST}"
echo "Signed ${REPOSITORY}@${DIGEST}"
