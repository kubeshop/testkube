#!/bin/bash
# OpenAPI Model Post-Processing Script
# 
# This script performs necessary transformations on generated OpenAPI models
# to ensure compatibility with the Testkube codebase.
#
# Usage: Called automatically by 'make generate-openapi'

set -euo pipefail

# ==================== Configuration ====================
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
readonly TMP_API_DIR="${PROJECT_ROOT}/tmp/api/testkube"
readonly TARGET_DIR="${PROJECT_ROOT}/pkg/api/v1/testkube"

# Detect OS for platform-specific sed commands
# Note: We'll use a function instead of a variable to handle sed portability
sed_inplace() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "$@"
    else
        sed -i "$@"
    fi
}

# ==================== Functions ====================
log_info() {
    echo "[INFO] $1"
}

log_error() {
    echo "[ERROR] $1" >&2
}

log_success() {
    echo "[SUCCESS] $1"
}

# Validate prerequisites
validate_environment() {
    if [[ ! -d "${TMP_API_DIR}" ]]; then
        log_error "Generated API directory not found: ${TMP_API_DIR}"
        log_error "Please run swagger-codegen first"
        exit 1
    fi
    
    if ! command -v sed &> /dev/null; then
        log_error "sed command not found"
        exit 1
    fi
}

# Move generated files to target directory
move_generated_files() {
    log_info "Moving generated files to target directory..."
    
    # Rename test files to avoid conflicts
    if [[ -f "${TMP_API_DIR}/model_test.go" ]]; then
        mv "${TMP_API_DIR}/model_test.go" "${TMP_API_DIR}/model_test_base.go" || true
    fi
    
    if [[ -f "${TMP_API_DIR}/model_test_suite_step_execute_test.go" ]]; then
        mv "${TMP_API_DIR}/model_test_suite_step_execute_test.go" \
           "${TMP_API_DIR}/model_test_suite_step_execute_test_base.go" || true
    fi
    
    # Create target directory if it doesn't exist
    mkdir -p "${TARGET_DIR}"
    
    # Move all model files
    find "${TMP_API_DIR}" -name "model_*.go" -exec mv {} "${TARGET_DIR}/" \;
    
    # Clean up temporary directory
    rm -rf "${PROJECT_ROOT}/tmp"
}

# Apply transformations to generated files
apply_transformations() {
    log_info "Applying transformations to generated files..."
    
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

# Apply update-specific transformations
apply_update_transformations() {
    log_info "Applying update-specific transformations..."
    
    # Make fields in update structs pointers
    find "${TARGET_DIR}" -name "*update*.go" -type f | while read -r file; do
        sed_inplace "s/ map/ \*map/g" "$file"
        sed_inplace "s/ string/ \*string/g" "$file"
        sed_inplace "s/ \[\]/ \*\[\]/g" "$file"
        sed_inplace "s/ int32/ \*int32/g" "$file"
        sed_inplace "s/ int64/ \*int64/g" "$file"
        sed_inplace "s/ bool/ \*bool/g" "$file"
        
        # Fix specific struct pointer types
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

# Apply special case transformations
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
    find "${TARGET_DIR}" -name "**_key_ref.go" -type f | while read -r file; do
        sed_inplace "s/ bool/ \*bool/g" "$file"
    done
}

# Main execution
main() {
    log_info "Starting OpenAPI post-processing..."
    
    # Validate environment
    validate_environment
    
    # Move generated files
    move_generated_files
    
    # Apply transformations
    apply_transformations
    apply_update_transformations
    apply_special_transformations
    
    log_success "OpenAPI post-processing completed successfully"
}

# Run main function
main "$@"