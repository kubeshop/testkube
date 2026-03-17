#!/bin/bash
# Generate OpenAPI Go models from testkube.yaml using swagger-codegen via Docker.
#
# Requires: docker
# Usage: Called via 'go generate ./pkg/api/v1/testkube' or 'make generate-openapi'

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
readonly TMP_DIR="${PROJECT_ROOT}/tmp"
readonly TMP_API_DIR="${TMP_DIR}/api/testkube"
readonly TARGET_DIR="${PROJECT_ROOT}/pkg/api/v1/testkube"

readonly SWAGGER_CODEGEN_IMAGE="swaggerapi/swagger-codegen-cli-v3:3.0.78"

# ==================== Helpers ====================

is_gnu_sed() {
    sed --version 2>/dev/null | grep -q "GNU sed"
}

sed_inplace() {
    if is_gnu_sed; then
        sed -i "$@"
    else
        sed -i '' "$@"
    fi
}

log_info()    { echo "[INFO] $1"; }
log_error()   { echo "[ERROR] $1" >&2; }
log_success() { echo "[SUCCESS] $1"; }

# ==================== Step 1: Run swagger-codegen via Docker ====================

run_swagger_codegen() {
    log_info "Running swagger-codegen via Docker..."

    if ! command -v docker &> /dev/null; then
        log_error "docker is not installed"
        exit 1
    fi

    rm -rf "${TMP_DIR}/api"

    docker run --rm \
        --user "$(id -u):$(id -g)" \
        -v "${PROJECT_ROOT}:/work" \
        -w /work \
        "${SWAGGER_CODEGEN_IMAGE}" \
        generate \
        --model-package testkube \
        -i api/v1/testkube.yaml \
        -l go \
        -o /work/tmp/api/testkube
}

# ==================== Step 2: Post-process generated files ====================

move_generated_files() {
    log_info "Moving generated model files..."

    if [[ ! -d "${TMP_API_DIR}" ]]; then
        log_error "Generated API directory not found: ${TMP_API_DIR}"
        exit 1
    fi

    # Rename test files to avoid conflicts
    if [[ -f "${TMP_API_DIR}/model_test.go" ]]; then
        mv "${TMP_API_DIR}/model_test.go" "${TMP_API_DIR}/model_test_base.go" || true
    fi
    if [[ -f "${TMP_API_DIR}/model_test_suite_step_execute_test.go" ]]; then
        mv "${TMP_API_DIR}/model_test_suite_step_execute_test.go" \
           "${TMP_API_DIR}/model_test_suite_step_execute_test_base.go" || true
    fi

    mkdir -p "${TARGET_DIR}"
    find "${TMP_API_DIR}" -name "model_*.go" -exec mv {} "${TARGET_DIR}/" \;
    rm -rf "${TMP_DIR}"
}

apply_transformations() {
    log_info "Applying transformations..."

    # Change package name from swagger to testkube
    find "${TARGET_DIR}" -type f -name "*.go" | while read -r file; do
        sed_inplace "s/package swagger/package testkube/g" "$file"
    done

    # Fix map pointer syntax
    find "${TARGET_DIR}" -type f -name "*.go" | while read -r file; do
        sed_inplace "s/\*map\[string\]/map[string]/g" "$file"
    done

    # Support map with empty additional properties
    find "${TARGET_DIR}" -name "*.go" -type f | while read -r file; do
        sed_inplace "s/ map\[string\]Object / map\[string\]interface{} /g" "$file"
    done

    # Support list with unknown values
    find "${TARGET_DIR}" -name "*.go" -type f | while read -r file; do
        sed_inplace "s/ \[\]Object / \[\]interface{} /g" "$file"
    done
}

apply_update_transformations() {
    log_info "Applying update-specific transformations..."

    find "${TARGET_DIR}" -name "*update*.go" -type f | while read -r file; do
        sed_inplace "s/ map/ \*map/g" "$file"
        sed_inplace "s/ string/ \*string/g" "$file"
        sed_inplace "s/ \[\]/ \*\[\]/g" "$file"
        sed_inplace "s/ int32/ \*int32/g" "$file"
        sed_inplace "s/ int64/ \*int64/g" "$file"
        sed_inplace "s/ bool/ \*bool/g" "$file"

        sed_inplace "s/ \*TestContent/ \*\*TestContentUpdate/g" "$file"
        sed_inplace "s/ \*ExecutionRequest/ \*\*ExecutionUpdateRequest/g" "$file"
        sed_inplace "s/ \*Repository/ \*\*RepositoryUpdate/g" "$file"
        sed_inplace "s/ \*SecretRef/ \*\*SecretRef/g" "$file"
        sed_inplace "s/ \*ArtifactRequest/ \*\*ArtifactUpdateRequest/g" "$file"
        sed_inplace "s/ \*TestSuiteExecutionRequest/ \*\*TestSuiteExecutionUpdateRequest/g" "$file"
        sed_inplace "s/ \*ExecutorMeta/ \*\*ExecutorMetaUpdate/g" "$file"
        sed_inplace "s/ \*PodRequest/ \*\*PodUpdateRequest/g" "$file"
        sed_inplace "s/ \*PodResourcesRequest/ \*\*PodResourcesUpdateRequest/g" "$file"
        sed_inplace "s/ \*ResourceRequest/ \*ResourceUpdateRequest/g" "$file"
        sed_inplace "s/ \*WebhookTemplateRef/ \*\*WebhookTemplateRef/g" "$file"
    done
}

apply_special_transformations() {
    log_info "Applying special case transformations..."

    # Add newline before Deprecated comments
    find "${TARGET_DIR}" -type f -name "*.go" | while read -r file; do
        sed_inplace "s/ Deprecated/ \\n\/\/ Deprecated/g" "$file"
    done

    # Make bool fields pointers in env source files
    find "${TARGET_DIR}" -name "*_env_source.go" -type f | while read -r file; do
        sed_inplace "s/ bool/ \*bool/g" "$file"
    done

    # Make bool fields pointers in key ref files
    find "${TARGET_DIR}" -name "*_key_ref.go" -type f | while read -r file; do
        sed_inplace "s/ bool/ \*bool/g" "$file"
    done
}

# ==================== Main ====================

main() {
    log_info "Generating OpenAPI models..."
    run_swagger_codegen
    move_generated_files
    apply_transformations
    apply_update_transformations
    apply_special_transformations
    cd "${PROJECT_ROOT}" && go fmt pkg/api/v1/testkube/*.go > /dev/null
    log_success "OpenAPI model generation complete"
}

main "$@"
