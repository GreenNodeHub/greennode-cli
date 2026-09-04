#!/usr/bin/env bash
# GreenNode CLI installer (macOS / Linux).
# Usage: curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.sh | bash
#
# Honors:
#   GRN_INSTALL_BASE_URL  (default https://github.com/GreenNodeHub/greennode-cli)
#   GRN_INSTALL_TAG       (default: resolve latest via /releases/latest 302)
#   GRN_INSTALL_ALLOW_SUDO (set to 1 to allow running under sudo from a regular user)
set -euo pipefail

BASE="${GRN_INSTALL_BASE_URL:-https://github.com/GreenNodeHub/greennode-cli}"

die() { echo "Error: $*" >&2; exit 1; }

# --- Step 1: sudo guard ---
if [ "$(id -u)" -eq 0 ] && [ -n "${SUDO_USER:-}" ] && [ "${GRN_INSTALL_ALLOW_SUDO:-0}" != "1" ]; then
  die "running under sudo would install into /root — set GRN_INSTALL_ALLOW_SUDO=1 to allow, or run without sudo."
fi

# --- Step 2: platform detect ---
OS="$(uname -s)"
ARCH="$(uname -m)"
case "$ARCH" in
  arm64|aarch64) ARCH=arm64 ;;
  x86_64|amd64)  ARCH=amd64 ;;
  *) die "unsupported arch: $ARCH" ;;
esac
case "$OS" in
  Darwin)
    # Rosetta-2 translation: flip amd64 -> arm64 under translation.
    if [ "$ARCH" = "amd64" ] && [ "$(sysctl -n sysctl.proc_translated 2>/dev/null || echo 0)" = "1" ]; then
      ARCH=arm64
    fi
    PLATFORM="darwin-${ARCH}"
    ;;
  Linux) PLATFORM="linux-${ARCH}" ;;
  *) die "unsupported OS: $OS" ;;
esac
# NOTE: no musl detection — binaries are static (CGO_ENABLED=0).

# --- Step 3: resolve latest tag (unless GRN_INSTALL_TAG set) ---
if [ -n "${GRN_INSTALL_TAG:-}" ]; then
  VTAG="$GRN_INSTALL_TAG"
else
  REDIRECT="$(curl -fsSL -o /dev/null -w '%{url_effective}' "${BASE}/releases/latest")"
  VTAG="${REDIRECT##*/tag/}"   # .../releases/tag/v1.12.0 -> v1.12.0
fi
case "$VTAG" in
  v[0-9]*) ;; # ok
  *) die "could not resolve a version tag from ${BASE}/releases/latest (got '${VTAG}') — set GRN_INSTALL_TAG manually." ;;
esac
TAG="${VTAG#v}"

# --- Step 4: fetch SHA256SUMS + extract expected hash for our platform ---
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
SUMS="$TMP/SHA256SUMS"
curl -fsSL "${BASE}/releases/download/${VTAG}/SHA256SUMS" -o "$SUMS" \
  || die "could not fetch SHA256SUMS for ${VTAG} from ${BASE} (check GRN_INSTALL_BASE_URL / GRN_INSTALL_TAG)."
# Match the exact platform binary line; -v${TAG} suffix anchors so amd64 != arm64.
EXPECTED="$(grep -E "grn-${PLATFORM}-${VTAG}(\.exe)?\$" "$SUMS" | awk '{print $1}' | head -1)"
[ -n "$EXPECTED" ] || die "no checksum line for ${PLATFORM} in SHA256SUMS — the release may not ship this platform."
# Sanity: it's 64 hex.
echo "$EXPECTED" | grep -qE '^[0-9a-f]{64}$' || die "malformed checksum for ${PLATFORM}: ${EXPECTED}"

# --- Step 5: download binary ---
BIN="$TMP/grn"
curl -fsSL "${BASE}/releases/download/${VTAG}/grn-${PLATFORM}-${VTAG}" -o "$BIN" \
  || die "could not fetch grn-${PLATFORM}-${VTAG} from ${BASE} (the release may not ship this platform; check GRN_INSTALL_BASE_URL / GRN_INSTALL_TAG)."

# --- Step 6: verify ---
if command -v sha256sum >/dev/null 2>&1; then HASHER=sha256sum; else HASHER="shasum -a 256"; fi
ACTUAL="$($HASHER "$BIN" | awk '{print $1}')"
[ "$ACTUAL" = "$EXPECTED" ] || { rm -f "$BIN"; die "checksum mismatch (expected ${EXPECTED}, got ${ACTUAL}) — the download may be corrupt or tampered."; }

# --- Step 7: install + PATH ---
INSTALL_DIR="$HOME/.local/lib/greennode"
BIN_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR" "$BIN_DIR"
mv "$BIN" "$INSTALL_DIR/grn"
chmod +x "$INSTALL_DIR/grn"
ln -sf "$INSTALL_DIR/grn" "$BIN_DIR/grn"

# Idempotent rc-PATH append: pick the right rc, grep-before-append.
rc_for_path() {
  case "$OS" in
    Darwin) for f in "$HOME/.zshrc" "$HOME/.zprofile"; do [ -f "$f" ] && echo "$f" && return; done; echo "$HOME/.zshrc" ;;
    Linux)  for f in "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile"; do [ -f "$f" ] && echo "$f" && return; done; echo "$HOME/.bashrc" ;;
  esac
}
RC="$(rc_for_path)"
# shellcheck disable=SC2016 # single quotes are intentional: this is a literal
# line to WRITE into the rc file so $HOME/$PATH expand at shell startup, not now.
PATH_LINE='export PATH="$HOME/.local/bin:$PATH"'
if ! grep -qF "$PATH_LINE" "$RC" 2>/dev/null; then
  printf '\n# Added by grn installer\n%s\n' "$PATH_LINE" >> "$RC"
fi

echo "Installed grn ${TAG} → ${BIN_DIR}/grn"
echo "Restart your shell, or run: export PATH=\"$HOME/.local/bin:\$PATH\""
