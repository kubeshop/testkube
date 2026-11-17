#!/bin/sh
# Served as: curl -fsSL https://trial.testkube.io | bash
# POSIX sh only, so dash and busybox run it too.
set -eu

RERUN="curl -fsSL https://trial.testkube.io | bash"
MIN_MINOR=15

main() {
  base_url="${TESTKUBE_BASE_URL:-https://github.com/kubeshop/testkube/releases}"
  bin_dir="$HOME/.testkube/bin"
  setup_colors

  printf 'Testkube trial installer\n'

  detect_system
  ok system "$os_label $arch"

  check_tools

  version=$(resolve_version)
  check_version "$version"
  ok version "$version"

  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM

  archive="testkube_${version}_${os}_${arch}.tar.gz"
  fix="Check your network, proxy or firewall, then run again:\n  $RERUN"
  if [ -n "${TESTKUBE_VERSION:-}" ]; then
    fix="Check that version $version exists at https://github.com/kubeshop/testkube/releases,\nor unset TESTKUBE_VERSION, then run again:\n  $RERUN"
  fi
  fetch "$base_url/download/$version/$archive" "$tmp/$archive" ||
    fail download "could not download $archive" "$fix"
  fetch "$base_url/download/$version/checksums.txt" "$tmp/checksums.txt" ||
    fail download "could not download checksums.txt" "$fix"
  verify_checksum "$tmp" "$archive"
  ok download "checksum verified"

  install_binary "$tmp" "$archive"
  # shellcheck disable=SC2088
  ok installed "~/.testkube/bin/testkube"

  path_hint
  start_installer
}

setup_colors() {
  green='' red='' reset=''
  if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    green=$(printf '\033[32m') red=$(printf '\033[31m') reset=$(printf '\033[0m')
  fi
}

ok() {
  printf '  %s✔%s %-10s %s\n' "$green" "$reset" "$1" "$2"
}

fail() {
  printf '  %s✖%s %-10s %s\n\n' "$red" "$reset" "$1" "$2" >&2
  printf '%b\n' "$3" >&2
  exit 1
}

detect_system() {
  case "$(uname -s)" in
    Darwin) os=Darwin os_label=macOS ;;
    Linux) os=Linux os_label=Linux ;;
    MINGW* | MSYS* | CYGWIN*)
      fail system "Windows is not supported here" "In your WSL terminal, run:\n  $RERUN"
      ;;
    *)
      fail system "$(uname -s) is not supported" "The trial runs on macOS and Linux."
      ;;
  esac
  # WSL 1 reports a capital M; WSL 2 a lowercase one.
  case "$(uname -r)" in
    *Microsoft*)
      fail system "WSL 1 can't run Docker" "Switch this distro to WSL 2. In PowerShell, run:\n  wsl --set-version ${WSL_DISTRO_NAME:-<distro>} 2\nThen run again:\n  $RERUN"
      ;;
  esac
  case "$(uname -m)" in
    x86_64 | amd64) arch=x86_64 ;;
    arm64 | aarch64) arch=arm64 ;;
    *)
      fail system "$(uname -m) is not supported" "The trial runs on x86_64 and arm64 machines."
      ;;
  esac
  # Rosetta shells report x86_64 on Apple silicon.
  if [ "$os" = Darwin ] && [ "$(/usr/sbin/sysctl -n sysctl.proc_translated 2>/dev/null)" = 1 ]; then
    arch=arm64
  fi
}

check_tools() {
  if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
    fail tools "needs curl or wget" "Install curl, then run again:\n  $RERUN"
  fi
  if ! command -v tar >/dev/null 2>&1; then
    fail tools "needs tar" "Install tar, then run again:\n  $RERUN"
  fi
  if ! command -v sha256sum >/dev/null 2>&1 && ! command -v shasum >/dev/null 2>&1; then
    fail tools "needs sha256sum or shasum" "Install coreutils, then run again:\n  $RERUN"
  fi
}

# /latest redirects to the newest stable tag, no API limits.
resolve_version() {
  if [ -n "${TESTKUBE_VERSION:-}" ]; then
    printf '%s\n' "${TESTKUBE_VERSION#v}"
    return
  fi
  if command -v curl >/dev/null 2>&1; then
    url=$(curl -fsSIL --retry 3 -o /dev/null -w '%{url_effective}' "$base_url/latest" 2>/dev/null) || url=''
  else
    url=$(wget -S --spider "$base_url/latest" 2>&1 | sed -n 's/^ *[Ll]ocation: *//p' | tail -n 1 | tr -d '\r') || url=''
  fi
  case "$url" in
    */tag/*) ;;
    *) url='' ;;
  esac
  if [ -z "$url" ]; then
    fail version "could not find the latest version" "Check your network, proxy or firewall, then run again:\n  $RERUN"
  fi
  printf '%s\n' "${url##*/}"
}

# Older releases have no `testkube install local`.
check_version() {
  case "$1" in
    [0-9]*.[0-9]*.[0-9]*) ;;
    *)
      fail version "$1 is not a version" "Set TESTKUBE_VERSION like 2.15.0, or unset it, then run again:\n  $RERUN"
      ;;
  esac
  major=${1%%.*}
  minor=${1#*.}
  minor=${minor%%.*}
  if [ "$major" -gt 2 ] || { [ "$major" -eq 2 ] && [ "$minor" -ge "$MIN_MINOR" ]; }; then
    return
  fi
  if [ -n "${TESTKUBE_VERSION:-}" ]; then
    fail version "$1 is too old" "The trial needs Testkube 2.$MIN_MINOR.0 or newer. Unset TESTKUBE_VERSION, then run again:\n  $RERUN"
  fi
  fail version "$1 is too old" "The trial needs Testkube 2.$MIN_MINOR.0 or newer, which is not released yet."
}

# A local mirror for testing may be plain http.
fetch() {
  if ! command -v curl >/dev/null 2>&1; then
    wget -q -O "$2" "$1"
  elif [ -n "${TESTKUBE_BASE_URL:-}" ]; then
    curl -fsSL --retry 3 -o "$2" "$1"
  else
    curl -fsSL --proto '=https' --tlsv1.2 --retry 3 -o "$2" "$1"
  fi
}

verify_checksum() {
  expected=$(awk -v f="$2" '$2 == f { print $1 }' "$1/checksums.txt")
  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$1/$2" | awk '{ print $1 }')
  else
    actual=$(shasum -a 256 "$1/$2" | awk '{ print $1 }')
  fi
  if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
    fail download "checksum does not match" "The download may be damaged. Run again:\n  $RERUN\nIf it fails again, contact us: https://testkube.io/contact"
  fi
}

install_binary() {
  tar -xzf "$1/$2" -C "$1" kubectl-testkube ||
    fail installed "could not unpack $2" "Run again:\n  $RERUN\nIf it fails again, contact us: https://testkube.io/contact"
  # Rename last, so a failed copy leaves no half binary.
  if ! mkdir -p "$bin_dir" ||
    ! cp "$1/kubectl-testkube" "$bin_dir/.kubectl-testkube.new" ||
    ! chmod 755 "$bin_dir/.kubectl-testkube.new" ||
    ! mv -f "$bin_dir/.kubectl-testkube.new" "$bin_dir/kubectl-testkube" ||
    ! ln -sf kubectl-testkube "$bin_dir/testkube"; then
    fail installed "could not write to ~/.testkube/bin" "Check that you can write to ~/.testkube/bin, then run again:\n  $RERUN"
  fi
  # Catches noexec home folders before the handoff fails oddly.
  if ! "$bin_dir/testkube" --help >/dev/null 2>&1; then
    fail installed "installed, but it does not run" "Check that programs can run from ~/.testkube/bin, then run again:\n  $RERUN"
  fi
}

path_hint() {
  found=$(command -v testkube 2>/dev/null || true)
  if [ "$found" = "$bin_dir/testkube" ]; then
    return
  fi
  if [ -n "$found" ]; then
    printf '\nAnother testkube comes first on your PATH: %s\n' "$found"
    printf 'To use this one from any terminal, add this to your shell profile:\n'
  else
    printf '\nTo use testkube from any terminal, add this to your shell profile:\n'
  fi
  # shellcheck disable=SC2016
  printf '  export PATH="$HOME/.testkube/bin:$PATH"\n'
}

# Under curl | bash, stdin is the script, not the keyboard.
start_installer() {
  printf '\nStarting the installer...\n\n'
  rm -rf "$tmp"
  trap - EXIT
  if (: </dev/tty) 2>/dev/null; then
    exec "$bin_dir/testkube" install local </dev/tty
  fi
  exec "$bin_dir/testkube" install local </dev/null
}

main "$@"
