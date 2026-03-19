#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HELM_DIR="${ROOT_DIR}/k8s/helm"

libraries=(
  global
  testkube-crds
)

consumers=(
  testkube-operator
  testkube-runner
)

status=0

for consumer in "${consumers[@]}"; do
  for library in "${libraries[@]}"; do
    source_dir="${HELM_DIR}/${library}"
    vendored_dir="${HELM_DIR}/${consumer}/charts/${library}"

    if [[ ! -d "${vendored_dir}" ]]; then
      echo "Missing vendored Helm library: ${vendored_dir}"
      status=1
      continue
    fi

    if ! diff -qr "${source_dir}" "${vendored_dir}" >/dev/null; then
      echo "Vendored Helm library is out of sync: ${consumer}/charts/${library}"
      echo "Run: make sync-helm-libraries"
      status=1
    fi
  done
done

exit "${status}"
