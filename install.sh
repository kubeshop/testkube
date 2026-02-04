#!/bin/bash
set -eo pipefail

echo "Getting kubectl-testkube plugin"

if [ ! -z "${DEBUG}" ];
then set -x
fi

_check_required_tools() {
  local MISSING_TOOLS=""
  for CMD in curl jq; do
    if ! which "${CMD}" &>/dev/null; then
      MISSING_TOOLS="${MISSING_TOOLS}${CMD} "
    fi
  done

  if [[ ${MISSING_TOOLS} != "" ]]; then
    echo "Missing required tools: ${MISSING_TOOLS}"
    echo Please install these using your package manager and try again.
    exit 1
  fi
}

_detect_arch() {
    case $(uname -m) in
    amd64|x86_64) echo "x86_64"
    ;;
    arm64|aarch64) echo "arm64"
    ;;
    i386) echo "i386"
    ;;
    *) echo "Unsupported processor architecture";
    return 1
    ;;
     esac
}

_detect_os(){
    case $(uname) in
    Linux) echo "Linux"
    ;;
    Darwin) echo "Darwin"
    ;;
    Windows) echo "Windows"
    ;;
     esac
}

_download_url() {
  local arch
  local os
  local tag
  local version

  arch="$(_detect_arch)"
  os="$(_detect_os)"
  if [ -z "$TESTKUBE_VERSION" ]; then
    if [ "$1" = "beta" ]; then
      tag="$(
        curl -s "https://api.github.com/repos/kubeshop/testkube/releases" \
        2>/dev/null \
        | jq -r '.[].tag_name | select(test("beta"))' \
        | head -n 1 \
      )"
        if [ -z "$tag" ]; then
            echo "No beta releases found. Installing latest release" >&2
            tag="$(
              curl -s "https://api.github.com/repos/kubeshop/testkube/releases/latest" \
              2>/dev/null \
              | jq -r '.tag_name' \
            )"
        fi
    else
      tag="$(
        curl -s "https://api.github.com/repos/kubeshop/testkube/releases/latest" \
        2>/dev/null \
        | jq -r '.tag_name' \
      )"
    fi
  else
    tag="$TESTKUBE_VERSION"
  fi
  version="${tag/#v/}" # remove leading v if present

  echo "https://github.com/kubeshop/testkube/releases/download/${tag}/testkube_${version:-1}_${os}_$arch.tar.gz"
}

_check_required_tools

if [ "$1" = "beta" ]; then
  url="$(_download_url "beta")"
  echo "Downloading testkube from URL: $url"
  curl -sSLf "$url" > testkube.tar.gz
else
  echo "Downloading testkube from URL: $(_download_url)"
  curl -sSLf "$(_download_url)" > testkube.tar.gz
fi

INSTALL_DIR=/usr/local/bin

echo "Installing testkube into ${INSTALL_DIR}"
INSTALL_PREFIX=""
if ! [[ -w "$INSTALL_DIR" ]]; then
  echo -e "\e[1;38;5;208m"
  echo "Looks like the current user does not have write access to ${INSTALL_DIR}"
  echo "You might be prompted to enter your password below by sudo"
  echo -e "\e[0m"
  INSTALL_PREFIX=sudo
fi

tar -xzf testkube.tar.gz kubectl-testkube
rm testkube.tar.gz
${INSTALL_PREFIX} mv kubectl-testkube ${INSTALL_DIR}/kubectl-testkube
${INSTALL_PREFIX} ln -sf ${INSTALL_DIR}/kubectl-testkube ${INSTALL_DIR}/testkube
${INSTALL_PREFIX} ln -sf ${INSTALL_DIR}/kubectl-testkube ${INSTALL_DIR}/tk

echo "kubectl-testkube installed in:"
echo "- ${INSTALL_DIR}/kubectl-testkube"
echo "- ${INSTALL_DIR}/testkube"
echo "- ${INSTALL_DIR}/tk"
echo ""

if ! which helm &>/dev/null || ! which kubectl &>/dev/null; then
  echo "You'll also need to install \`helm\` and \`kubectl\`."
  echo "- Install Helm: https://helm.sh/docs/intro/install/"
  echo "- Install kubectl: https://kubernetes.io/docs/tasks/tools/#kubectl"
fi