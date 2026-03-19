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

for consumer in "${consumers[@]}"; do
  consumer_charts_dir="${HELM_DIR}/${consumer}/charts"
  mkdir -p "${consumer_charts_dir}"

  for library in "${libraries[@]}"; do
    src_dir="${HELM_DIR}/${library}"
    dest_dir="${consumer_charts_dir}/${library}"

    rm -rf "${dest_dir}"
    cp -R "${src_dir}" "${dest_dir}"
  done
done
