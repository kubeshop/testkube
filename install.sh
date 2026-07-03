#!/bin/sh
# POSIX sh compatible — this script is documented as `curl -sSLf https://get.testkube.io | sh`,
# so it must not rely on bash-only constructs ([[ ]], =~, set -o pipefail, etc.).
set -e

echo "Getting kubectl-testkube plugin"

if [ -n "${DEBUG}" ]; then
  set -x
fi

GITHUB_API="https://api.github.com/repos/kubeshop/testkube"
RELEASES_BASE_URL="https://github.com/kubeshop/testkube/releases/download"
INSTALL_DIR=/usr/local/bin

_check_required_tools() {
  MISSING_TOOLS=""
  for CMD in curl jq tar; do
    if ! command -v "${CMD}" >/dev/null 2>&1; then
      MISSING_TOOLS="${MISSING_TOOLS}${CMD} "
    fi
  done

  if [ -n "${MISSING_TOOLS}" ]; then
    echo "Missing required tools: ${MISSING_TOOLS}" >&2
    echo "Please install these using your package manager and try again." >&2
    exit 1
  fi
}

_detect_arch() {
  case $(uname -m) in
  amd64 | x86_64) echo "x86_64" ;;
  arm64 | aarch64) echo "arm64" ;;
  *)
    echo "Unsupported processor architecture: $(uname -m)" >&2
    return 1
    ;;
  esac
}

_detect_os() {
  case $(uname) in
  Linux) echo "Linux" ;;
  Darwin) echo "Darwin" ;;
  *)
    echo "Unsupported operating system: $(uname)" >&2
    echo "On Windows, download testkube from https://github.com/kubeshop/testkube/releases/latest" >&2
    return 1
    ;;
  esac
}

_resolve_tag() {
  if [ -n "${TESTKUBE_VERSION}" ]; then
    echo "${TESTKUBE_VERSION}"
    return 0
  fi

  if [ "$1" = "beta" ]; then
    TAG="$(
      curl -sSf "${GITHUB_API}/releases" 2>/dev/null |
        jq -r '[.[] | select(.prerelease)][0].tag_name // empty'
    )"
    if [ -n "${TAG}" ]; then
      echo "${TAG}"
      return 0
    fi
    echo "No pre-releases found. Installing latest release" >&2
  fi

  curl -sSf "${GITHUB_API}/releases/latest" 2>/dev/null | jq -r '.tag_name // empty'
}

# Normalize the tag format based on the version number when a version is
# explicitly provided. Starting from 2.4.0, release tags dropped the 'v'
# prefix. For auto-detected versions fetched from the API, the tag already
# has the correct format.
_normalize_tag() {
  TAG="$1"
  VERSION="$2"
  if [ -z "${TESTKUBE_VERSION}" ]; then
    echo "${TAG}"
    return 0
  fi

  # Only attempt numeric comparison when the version looks like a full numeric
  # semver (X.Y.Z). Strip any pre-release/build suffix first.
  BASE_VERSION="${VERSION%%-*}"
  if echo "${BASE_VERSION}" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    MAJOR="$(echo "${BASE_VERSION}" | cut -d. -f1)"
    MINOR="$(echo "${BASE_VERSION}" | cut -d. -f2)"
    if [ "${MAJOR}" -lt 2 ] || { [ "${MAJOR}" -eq 2 ] && [ "${MINOR}" -lt 4 ]; }; then
      echo "v${VERSION}"
      return 0
    fi
    echo "${VERSION}"
    return 0
  fi
  echo "${TAG}"
}

# Verify the downloaded tarball against the release's checksums.txt.
# Skips with a warning when no sha256 tool is available or the checksum entry
# is missing; fails hard on an actual mismatch.
_verify_checksum() {
  TARBALL_PATH="$1"
  TARBALL_NAME="$2"

  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL="$(sha256sum "${TARBALL_PATH}" | cut -d' ' -f1)"
  elif command -v shasum >/dev/null 2>&1; then
    ACTUAL="$(shasum -a 256 "${TARBALL_PATH}" | cut -d' ' -f1)"
  else
    echo "Warning: no sha256sum or shasum tool found, skipping checksum verification" >&2
    return 0
  fi

  if ! curl -sSLf "${RELEASES_BASE_URL}/${TAG}/checksums.txt" >"${WORKDIR}/checksums.txt" 2>/dev/null; then
    echo "Warning: could not download checksums.txt, skipping checksum verification" >&2
    return 0
  fi

  EXPECTED="$(grep " ${TARBALL_NAME}\$" "${WORKDIR}/checksums.txt" | cut -d' ' -f1)"
  if [ -z "${EXPECTED}" ]; then
    echo "Warning: no checksum entry found for ${TARBALL_NAME}, skipping checksum verification" >&2
    return 0
  fi

  if [ "${ACTUAL}" != "${EXPECTED}" ]; then
    echo "Checksum verification failed for ${TARBALL_NAME}" >&2
    echo "Expected: ${EXPECTED}" >&2
    echo "Actual:   ${ACTUAL}" >&2
    exit 1
  fi
  echo "Checksum verified"
}

_check_required_tools

ARCH="$(_detect_arch)"
OS="$(_detect_os)"

TAG="$(_resolve_tag "$1")"
if [ -z "${TAG}" ]; then
  echo "Could not determine the Testkube version to install." >&2
  echo "Set TESTKUBE_VERSION explicitly and try again, e.g. TESTKUBE_VERSION=2.11.0" >&2
  exit 1
fi

VERSION="${TAG#v}" # remove leading v if present
TAG="$(_normalize_tag "${TAG}" "${VERSION}")"

TARBALL_NAME="testkube_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="${RELEASES_BASE_URL}/${TAG}/${TARBALL_NAME}"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "${WORKDIR}"' EXIT

echo "Downloading testkube from URL: ${URL}"
curl -sSLf "${URL}" >"${WORKDIR}/testkube.tar.gz"

_verify_checksum "${WORKDIR}/testkube.tar.gz" "${TARBALL_NAME}"

tar -xzf "${WORKDIR}/testkube.tar.gz" -C "${WORKDIR}" kubectl-testkube

echo "Installing testkube into ${INSTALL_DIR}"
INSTALL_PREFIX=""
if [ ! -w "${INSTALL_DIR}" ]; then
  printf '\033[1;38;5;208m\n'
  echo "Looks like the current user does not have write access to ${INSTALL_DIR}"
  echo "You might be prompted to enter your password below by sudo"
  printf '\033[0m\n'
  INSTALL_PREFIX=sudo
fi

${INSTALL_PREFIX} mv "${WORKDIR}/kubectl-testkube" "${INSTALL_DIR}/kubectl-testkube"
${INSTALL_PREFIX} ln -sf "${INSTALL_DIR}/kubectl-testkube" "${INSTALL_DIR}/testkube"
${INSTALL_PREFIX} ln -sf "${INSTALL_DIR}/kubectl-testkube" "${INSTALL_DIR}/tk"

echo "kubectl-testkube installed in:"
echo "- ${INSTALL_DIR}/kubectl-testkube"
echo "- ${INSTALL_DIR}/testkube"
echo "- ${INSTALL_DIR}/tk"
echo ""

if ! command -v helm >/dev/null 2>&1 || ! command -v kubectl >/dev/null 2>&1; then
  echo "You'll also need to install \`helm\` and \`kubectl\`."
  echo "- Install Helm: https://helm.sh/docs/intro/install/"
  echo "- Install kubectl: https://kubernetes.io/docs/tasks/tools/#kubectl"
fi
