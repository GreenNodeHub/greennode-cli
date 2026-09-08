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

## Gating a service that is still behind a build tag

`grn vlb` is registered behind `//go:build !vks_only` (`go/cmd/register_vlb.go`), so it is built and tested in CI but excluded from the release binary, which `release.yml` builds with `-tags vks_only`. The same applies to `grn vserver`.

**Before removing that tag for a service, its live E2E runbook must have been run end to end against a real project**, because the Go tests cannot reach the `RunE` network path (resolve → request → output) — `cli.NewTokenProvider` only targets the prod/dev IAM presets, so a command cannot be pointed at a local server without real credentials.

- [ ] **VLB:** run [`tests/vlb/RUNBOOK.md`](tests/vlb/RUNBOOK.md) in full (~2 hours, creates 3–4 real load balancers). Fill in `tests/vlb/RESULTS.md` from `RESULTS.template.md` and attach it to the PR that removes the tag. All six P0 regressions must be green, teardown must be clean, and the credentials used for the run must be rotated afterwards.

Any real bug the runbook finds gets fixed, pinned by a unit test, and documented in the service's `CLAUDE.md` plus `docs/commands/<service>/` before the tag comes off — recording it in `RESULTS.md` is not enough.
