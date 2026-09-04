#!/usr/bin/env bash
# GreenNode CLI uninstaller (macOS / Linux).
# Removes what scripts/install.sh put down: the ~/.local/lib/greennode binary
# dir, the ~/.local/bin/grn symlink, and the PATH line from your shell rc.
#
# Usage: curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/uninstall.sh | bash
#   bash uninstall.sh --purge   # also remove ~/.greennode config + credentials
#
# Honors:
#   GRN_INSTALL_ALLOW_SUDO (set to 1 to allow running under sudo from a regular user)
set -euo pipefail

PURGE=0
for arg in "$@"; do
  case "$arg" in
    --purge) PURGE=1 ;;
    -h|--help)
      sed -n '2,9p' "$0"
      exit 0
      ;;
    *) echo "Error: unknown arg: $arg (try --help)" >&2; exit 2 ;;
  esac
done

die() { echo "Error: $*" >&2; exit 1; }

# --- Step 1: sudo guard (same as install.sh) ---
if [ "$(id -u)" -eq 0 ] && [ -n "${SUDO_USER:-}" ] && [ "${GRN_INSTALL_ALLOW_SUDO:-0}" != "1" ]; then
  die "running under sudo would remove from /root — set GRN_INSTALL_ALLOW_SUDO=1 to allow, or run without sudo."
fi

# --- Step 2: platform detect (need OS for rc selection) ---
OS="$(uname -s)"
case "$OS" in
  Darwin|Linux) ;;
  *) die "unsupported OS: $OS (this uninstaller is for macOS / Linux)" ;;
esac

# --- Step 3: remove binary dir + symlink ---
INSTALL_DIR="$HOME/.local/lib/greennode"
BIN_DIR="$HOME/.local/bin"
rm -f "$BIN_DIR/grn" 2>/dev/null || true
rm -rf "$INSTALL_DIR" 2>/dev/null || true

# --- Step 4: remove PATH line from rc (idempotent) ---
rc_for_path() {
  case "$OS" in
    Darwin) for f in "$HOME/.zshrc" "$HOME/.zprofile"; do [ -f "$f" ] && echo "$f" && return; done; echo "$HOME/.zshrc" ;;
    Linux)  for f in "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile"; do [ -f "$f" ] && echo "$f" && return; done; echo "$HOME/.bashrc" ;;
  esac
}
RC="$(rc_for_path)"
# shellcheck disable=SC2016 # single quotes are intentional: this is the literal
# line install.sh writes, so we match it byte-for-byte (expands at shell startup).
PATH_LINE='export PATH="$HOME/.local/bin:$PATH"'
if [ -f "$RC" ] && grep -qF "$PATH_LINE" "$RC" 2>/dev/null; then
  # Remove the PATH line + its "# Added by grn installer" comment. A leading
  # blank line from the installer's printf is left behind (harmless). grep -v
  # outputs every non-matching line; mktemp+mv (not sed -i) for BSD/macOS.
  tmp_rc="$(mktemp)"
  grep -vF -e "$PATH_LINE" -e "# Added by grn installer" "$RC" > "$tmp_rc" || true
  mv "$tmp_rc" "$RC"
fi

# --- Step 5: --purge config + credentials ---
if [ "$PURGE" -eq 1 ]; then
  rm -rf "$HOME/.greennode" 2>/dev/null || true
  rm -rf "$HOME/.greenode" 2>/dev/null || true   # legacy pre-rename config dir
fi

echo "Uninstalled grn"
if [ "$PURGE" -eq 1 ]; then
  echo "Also removed ~/.greennode/ (config + credentials)"
fi
echo "Restart your shell, or open a new terminal, for PATH changes to take effect."
