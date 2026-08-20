#!/usr/bin/env bash
# Hermetic test for scripts/uninstall.sh.
#
# Creates the install state (binary dir + symlink + rc PATH line) in a temp HOME,
# runs the uninstaller, asserts everything is gone — then runs again (idempotency)
# and tests --purge. No HTTP server needed: uninstall is pure local file ops.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT="$REPO_ROOT/scripts/uninstall.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

export HOME="$TMP/home"
mkdir -p "$HOME"

# Force darwin-arm64 via stubbed uname (same as install.test.sh) so rc_for_path
# picks .zshrc deterministically. uninstall.sh only calls `uname -s`.
mkdir -p "$TMP/bin"
cat > "$TMP/bin/uname" <<'EOF'
#!/usr/bin/env bash
case "$1" in
  -s) echo Darwin;;
  -m) echo arm64;;
  *) /usr/bin/uname "$@";;
esac
EOF
chmod +x "$TMP/bin/uname"
export PATH="$TMP/bin:$PATH"

# --- Set up the install state (exactly what install.sh would have created) -----
INSTALL_DIR="$HOME/.local/lib/greennode"
BIN_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR" "$BIN_DIR"
printf 'fake grn\n' > "$INSTALL_DIR/grn"
chmod +x "$INSTALL_DIR/grn"
ln -sf "$INSTALL_DIR/grn" "$BIN_DIR/grn"
# rc: the exact 2-line block install.sh writes, plus a sentinel line to confirm
# the uninstaller removes only the grn lines, not neighboring content.
RC="$HOME/.zshrc"
printf '\n# Added by grn installer\nexport PATH="$HOME/.local/bin:$PATH"\n' >> "$RC"
printf 'export OTHER_TOOL=1\n' >> "$RC"

# --- Run 1: uninstall ---------------------------------------------------------
bash "$SCRIPT"

# --- Assertions after run 1 ---------------------------------------------------
[ ! -e "$BIN_DIR/grn" ]     || { echo "FAIL: ~/.local/bin/grn symlink still exists"; exit 1; }
[ ! -d "$INSTALL_DIR" ]     || { echo "FAIL: ~/.local/lib/greennode dir still exists"; exit 1; }
[ ! -f "$INSTALL_DIR/grn" ] || { echo "FAIL: binary still exists"; exit 1; }
if grep -qF 'export PATH="$HOME/.local/bin:$PATH"' "$RC"; then
  echo "FAIL: PATH line still in rc"; exit 1
fi
if grep -qF '# Added by grn installer' "$RC"; then
  echo "FAIL: installer comment still in rc"; exit 1
fi
grep -qF 'export OTHER_TOOL=1' "$RC" || { echo "FAIL: sentinel line was removed (over-removal)"; exit 1; }

# --- Run 2: idempotency (uninstall again — no-op, must exit 0) -----------------
bash "$SCRIPT"

# --- --purge ------------------------------------------------------------------
mkdir -p "$HOME/.greennode"
printf 'creds\n' > "$HOME/.greennode/credentials"
mkdir -p "$HOME/.greenode"   # legacy pre-rename dir
bash "$SCRIPT" --purge
[ ! -d "$HOME/.greennode" ] || { echo "FAIL: --purge did not remove ~/.greennode"; exit 1; }
[ ! -d "$HOME/.greenode" ]  || { echo "FAIL: --purge did not remove legacy ~/.greenode"; exit 1; }

echo "PASS: uninstall.sh removes binary + symlink + rc PATH line (idempotent, --purge works)"
