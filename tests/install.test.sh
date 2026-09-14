#!/usr/bin/env bash
# Hermetic end-to-end test for scripts/install.sh.
#
# Stands up a fake release behind a local HTTP server (plain python3 -m
# http.server — no GitHub 302 emulation needed), points the script at it via
# GRN_INSTALL_BASE_URL + GRN_INSTALL_TAG + a temp HOME, and asserts a correct,
# checksum-verified, idempotent install.
#
# GRN_INSTALL_TAG=v9.9.9 makes the script skip the real /releases/latest 302
# tag-resolve (which a static file server can't emulate). The script still
# implements the real tag-resolve path for production; only the test skips it.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT="$REPO_ROOT/scripts/install.sh"
TMP="$(mktemp -d)"
SRV_PID=""
trap 'rm -rf "$TMP"; [ -n "$SRV_PID" ] && kill "$SRV_PID" 2>/dev/null || true' EXIT

# --- Stand up the fake release tree -------------------------------------
# Served root: $TMP/srv  (so http://host/releases/download/v9.9.9/... works)
FAKE_NAME="grn-darwin-arm64-v9.9.9"
REL_DIR="$TMP/srv/releases/download/v9.9.9"
mkdir -p "$REL_DIR"

# Fake "binary" is a shell script so it runs without a real build; the installer
# only cares about the checksum, not the file type.
cat > "$REL_DIR/grn.sh" <<'EOFSH'
#!/usr/bin/env bash
echo "grn-cli/9.9.9 fake"
EOFSH
chmod +x "$REL_DIR/grn.sh"
cp "$REL_DIR/grn.sh" "$REL_DIR/$FAKE_NAME"

# Build SHA256SUMS (GNU format) using the SAME hasher the installer will use,
# so the expected and actual hashes match (macOS: shasum -a 256 fallback).
if command -v sha256sum >/dev/null 2>&1; then HASHER=sha256sum; else HASHER="shasum -a 256"; fi
HASH="$( cd "$REL_DIR" && $HASHER "$FAKE_NAME" | awk '{print $1}' )"
printf '%s  %s\n' "$HASH" "$FAKE_NAME" > "$REL_DIR/SHA256SUMS"

# --- Start a plain static HTTP server on a fixed port (retry if taken) ----
# We bind a fixed high port instead of parsing `python3 -m http.server`'s startup
# banner: macOS Python does not flush the "Serving HTTP on ... port N" line to the
# redirected log, so log-grep port discovery is unreliable across platforms. Binding
# a fixed port and probing with curl until it answers is robust.
SRV_LOG="$TMP/srv.log"
PORT=""
for candidate in 18080 18081 18082 18083 18084 18085; do
  python3 -m http.server "$candidate" --bind 127.0.0.1 --directory "$TMP/srv" >"$SRV_LOG" 2>&1 &
  SRV_PID=$!
  for _ in $(seq 1 20); do
    if curl -fsS "http://127.0.0.1:${candidate}/" >/dev/null 2>&1; then
      PORT="$candidate"
      break
    fi
    # If the server died immediately, the port was taken — try the next one.
    if ! kill -0 "$SRV_PID" 2>/dev/null; then break; fi
    sleep 0.1
  done
  [ -n "$PORT" ] && break
  kill "$SRV_PID" 2>/dev/null || true
  wait "$SRV_PID" 2>/dev/null || true
done
[ -n "$PORT" ] || { echo "FAIL: test HTTP server did not start on any candidate port"; cat "$SRV_LOG"; exit 1; }

# --- Point the installer at the fake release ----------------------------
export GRN_INSTALL_BASE_URL="http://127.0.0.1:${PORT}"
export GRN_INSTALL_TAG="v9.9.9"   # skip tag-resolve; pin the version
export HOME="$TMP/home"           # install into temp home, never real ~/.local
mkdir -p "$HOME"

# Force the platform to darwin-arm64 deterministically by stubbing uname/sysctl
# in a temp PATH dir. The fake binary name above matches this stubbed platform.
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
cat > "$TMP/bin/sysctl" <<'EOF'
#!/usr/bin/env bash
# proc_translated returns empty (not under Rosetta) — keep arm64.
echo ""
EOF
chmod +x "$TMP/bin/sysctl"
export PATH="$TMP/bin:$PATH"

# --- Run the installer (run 1) -----------------------------------------
bash "$SCRIPT"

# --- Assertions ---------------------------------------------------------
[ -x "$HOME/.local/lib/greennode/grn" ] || { echo "FAIL: binary not installed"; exit 1; }
[ -L "$HOME/.local/bin/grn" ] || { echo "FAIL: ~/.local/bin/grn symlink missing"; exit 1; }
[ "$("$HOME/.local/bin/grn" 2>&1)" = "grn-cli/9.9.9 fake" ] || { echo "FAIL: wrong binary landed at symlink target"; exit 1; }

# rc got exactly one PATH line after run 1; running again must NOT add a second
# (idempotency). Counts the literal PATH line the script appends.
RUN_COUNT=$(grep -cF 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.zshrc" 2>/dev/null || echo 0)
bash "$SCRIPT"
RUN_COUNT_2=$(grep -cF 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.zshrc" 2>/dev/null || echo 0)
[ "$RUN_COUNT" -eq 1 ] || { echo "FAIL: expected 1 PATH line after run 1, got $RUN_COUNT"; exit 1; }
[ "$RUN_COUNT_2" -eq 1 ] || { echo "FAIL: idempotency broken — expected 1 PATH line after run 2, got $RUN_COUNT_2"; exit 1; }

echo "PASS: install.sh installs verified binary + idempotent PATH"
