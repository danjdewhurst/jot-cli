#!/bin/sh
# install.sh — curl-pipe installer for jot-cli
# Usage: curl -fsSL https://raw.githubusercontent.com/danjdewhurst/jot-cli/main/install.sh | sh
#
# Environment variables:
#   VERSION      Pin a specific version (default: latest release)
#   INSTALL_DIR  Custom install location (default: ~/.local/bin)

set -eu

REPO="danjdewhurst/jot-cli"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# --- Helpers ---

log() {
  printf '%s\n' "$@"
}

fail() {
  log "Error: $1" >&2
  exit 1
}

need_cmd() {
  if ! command -v "$1" > /dev/null 2>&1; then
    fail "required command not found: $1"
  fi
}

# --- Detect OS and architecture ---

detect_platform() {
  os="$(uname -s)"
  arch="$(uname -m)"

  case "$os" in
    Darwin) os="darwin" ;;
    Linux)  os="linux" ;;
    *)      fail "unsupported operating system: $os" ;;
  esac

  case "$arch" in
    x86_64)  arch="amd64" ;;
    aarch64) arch="arm64" ;;
    arm64)   arch="arm64" ;;
    *)       fail "unsupported architecture: $arch" ;;
  esac

  PLATFORM_OS="$os"
  PLATFORM_ARCH="$arch"
}

# --- Resolve version ---

resolve_version() {
  if [ -n "${VERSION:-}" ]; then
    # Strip leading v if present
    VERSION="${VERSION#v}"
    return
  fi

  log "Fetching latest release..."
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"v?([^"]+)".*/\1/')

  if [ -z "$VERSION" ]; then
    fail "could not determine latest version"
  fi
}

# --- Download and verify ---

download_and_verify() {
  archive="jot-cli_${VERSION}_${PLATFORM_OS}_${PLATFORM_ARCH}.tar.gz"
  base_url="https://github.com/${REPO}/releases/download/v${VERSION}"

  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  log "Downloading jot-cli v${VERSION} (${PLATFORM_OS}/${PLATFORM_ARCH})..."
  curl -fsSL "${base_url}/${archive}" -o "${tmpdir}/${archive}"
  curl -fsSL "${base_url}/checksums.txt" -o "${tmpdir}/checksums.txt"

  log "Verifying checksum..."
  expected=$(grep "${archive}" "${tmpdir}/checksums.txt" | awk '{print $1}')
  if [ -z "$expected" ]; then
    fail "archive not found in checksums.txt"
  fi

  if command -v sha256sum > /dev/null 2>&1; then
    actual=$(sha256sum "${tmpdir}/${archive}" | awk '{print $1}')
  elif command -v shasum > /dev/null 2>&1; then
    actual=$(shasum -a 256 "${tmpdir}/${archive}" | awk '{print $1}')
  else
    fail "no sha256 tool found (need sha256sum or shasum)"
  fi

  if [ "$expected" != "$actual" ]; then
    fail "checksum mismatch: expected ${expected}, got ${actual}"
  fi

  log "Checksum verified."

  # Extract
  tar -xzf "${tmpdir}/${archive}" -C "${tmpdir}" jot-cli
  BINARY_PATH="${tmpdir}/jot-cli"
}

# --- Install ---

install_binary() {
  mkdir -p "$INSTALL_DIR"

  mv "$BINARY_PATH" "${INSTALL_DIR}/jot-cli"
  chmod +x "${INSTALL_DIR}/jot-cli"

  # Create j symlink
  ln -sf "${INSTALL_DIR}/jot-cli" "${INSTALL_DIR}/j"

  log ""
  log "jot-cli v${VERSION} installed to ${INSTALL_DIR}/jot-cli"
  log "Symlink: ${INSTALL_DIR}/j -> jot-cli"

  # Check if install dir is in PATH
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
      log ""
      log "Add ${INSTALL_DIR} to your PATH:"
      log "  export PATH=\"${INSTALL_DIR}:\$PATH\""
      ;;
  esac
}

# --- Main ---

main() {
  need_cmd curl
  need_cmd tar

  detect_platform
  resolve_version
  download_and_verify
  install_binary
}

main
