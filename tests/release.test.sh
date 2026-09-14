#!/usr/bin/env bash
# Verifies the SHA256SUMS generation step (Section 1 step 3 of the spec)
# produces exactly 6 GNU-format lines, one per versioned binary, no header.
#
# This is a contract test: it locks the shape of the CI command
# `sha256sum grn-*-v* > SHA256SUMS` (run in the release job on ubuntu-latest)
# and the GNU-format output it must produce. It synthesizes its own 6 fake
# versioned binaries in a temp dist/ and runs the exact command, so it passes
# without running CI by design.
set -euo pipefail

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# Synthesize 6 fake versioned binaries (content irrelevant; only names + hashes matter)
for plat in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 windows-arm64; do
  ext=""
  case "$plat" in windows-*) ext=".exe";; esac
  name="grn-${plat}-v1.12.0${ext}"
  printf 'fake binary for %s\n' "$name" > "$TMP/$name"
done

# The exact command the CI release job runs (ubuntu-latest has GNU sha256sum).
# On macOS developer machines that only have BSD shasum, fall back to it — it
# emits the same GNU-format output (64 hex + two spaces + filename).
SHA_CMD="sha256sum"
command -v sha256sum >/dev/null 2>&1 || SHA_CMD="shasum -a 256"

( cd "$TMP" && $SHA_CMD grn-*-v* > SHA256SUMS )

# Assertions
line_count=$(wc -l < "$TMP/SHA256SUMS" | tr -d ' ')
[ "$line_count" -eq 6 ] || { echo "FAIL: expected 6 lines, got $line_count"; cat "$TMP/SHA256SUMS"; exit 1; }

# Every line is GNU format: 64 hex + two spaces + a filename that ends in -v1.12.0[.exe]
while IFS= read -r line; do
  # 64 hex, two spaces, filename
  echo "$line" | grep -qE '^[0-9a-f]{64}  grn-(linux-(amd64|arm64)|darwin-(amd64|arm64)|windows-(amd64|arm64))-v1\.12\.0(\.exe)?$' \
    || { echo "FAIL: bad line format: $line"; exit 1; }
done < "$TMP/SHA256SUMS"

# No header / no blank lines: line 1 must be a real data line (covered above), and wc -l == 6 (no extras)
echo "PASS: SHA256SUMS has 6 correctly-formatted GNU lines"
