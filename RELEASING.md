# Releasing GreenNode CLI

Releases are automated by release-please (merge the `chore: release main` PR → tags `vX.Y.Z` → `release.yml` builds + attaches versioned binaries + `SHA256SUMS`). This checklist covers the **manual pre-release smoke** that the automated hermetic tests cannot run: the real one-liners against a real tagged release.

After a release tag `vX.Y.Z` exists (with binaries + `SHA256SUMS` attached), run each one-liner on a real machine:

- [ ] **macOS (arm64):** `curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.sh | bash` → `grn --version` prints `X.Y.Z`.
- [ ] **macOS (x86_64, or under Rosetta):** same one-liner → `grn --version` correct.
- [ ] **Linux (amd64):** same one-liner → `grn --version` correct.
- [ ] **Linux (arm64):** same one-liner → `grn --version` correct.
- [ ] **Windows (PowerShell, amd64):** `irm https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.ps1 | iex` → `grn --version` correct.
- [ ] **Windows (CMD, amd64, PowerShell-blocked box if available):** `curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.cmd -o install.cmd && install.cmd && del install.cmd` → `grn --version` correct.
- [ ] **Windows (arm64, if hardware available):** `install.ps1` → `grn --version` correct.

If a one-liner fails: check the release has 6 binaries + `SHA256SUMS`, the versioned filenames match `grn-<plat>-vX.Y.Z[.exe]`, and the `SHA256SUMS` line for the platform is present. Do not announce the release until all passing one-liners pass.
