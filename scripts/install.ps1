# GreenNode CLI installer (Windows PowerShell).
# Usage: irm https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.ps1 | iex
param(
  # Allow pinning a version instead of resolving latest (also used by tests).
  [string]$Tag
)
# Strict mode OFF by intent. Under Set-StrictMode -Version Latest, Windows
# PowerShell 5.1 (the irm|iex target) false-positives on $bin at Move-Item
# (Step 7): $bin is assigned at Step 5 and already consumed by Get-FileHash at
# Step 6 with no error, yet Move-Item -Path $bin throws "VariableIsUndefined",
# after which Move-Item prompts for the now-"missing" mandatory -Path and hangs
# ~2 min on a TTY-less CI stdin. Observed on Windows Server 2025 CI (PR #17).
# $ErrorActionPreference = "Stop" below still turns cmdlet errors into
# terminating errors, so fail-fast behavior is preserved without the quirk.
Set-StrictMode -Off
$ErrorActionPreference = "Stop"
$ProgressPreference = 'SilentlyContinue'

$BASE = $env:GRN_INSTALL_BASE_URL
if (-not $BASE) { $BASE = "https://github.com/GreenNodeHub/greennode-cli" }
if (-not $Tag) { $Tag = $env:GRN_INSTALL_TAG }

# --- Step 1: 64-bit guard ---
if (-not [Environment]::Is64BitProcess) {
  [Console]::Error.WriteLine("install.ps1: GreenNode CLI requires 64-bit Windows."); exit 1
}

# --- Step 2: arch detect ---
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { $platform = "windows-arm64" }
else { $platform = "windows-amd64" }

# --- Step 3: resolve tag ---
if ($Tag) {
  $vtag = $Tag
  if ($vtag -notlike "v*") { $vtag = "v$vtag" }
} else {
  # GitHub 302 /releases/latest -> /releases/tag/vX.Y.Z. Capture the Location.
  # -UseBasicParsing: PS 5.1 otherwise uses the IE COM engine, which is absent on
  # Server Core / IE-less Windows (a common CLI-install target) and would throw
  # "Internet Explorer engine is not available" on this HTML release page.
  $resp = Invoke-WebRequest -Uri "$BASE/releases/latest" -MaximumRedirection 0 -UseBasicParsing -ErrorAction SilentlyContinue
  if ($resp.StatusCode -eq 302 -and $resp.Headers.Location) {
    $loc = [string]$resp.Headers.Location
    $vtag = ($loc -split '/')[-1]   # last segment is v1.12.0
  } else {
    # Fallback: follow the redirect and read the effective URL.
    $eff = Invoke-WebRequest -Uri "$BASE/releases/latest" -MaximumRedirection 5 -UseBasicParsing
    $vtag = ($eff.BaseResponse.ResponseUri.AbsolutePath -split '/')[-1]
  }
}
if ($vtag -notmatch '^v\d+\.\d+\.\d+') {
  [Console]::Error.WriteLine("install.ps1: Could not resolve a version tag from $BASE/releases/latest (got '$vtag'). Set -Tag or GRN_INSTALL_TAG."); exit 1
}
$tag = $vtag.TrimStart('v')

# --- Step 4: fetch SHA256SUMS + expected hash ---
$dl = Join-Path $env:TEMP ("grn-install-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $dl | Out-Null
$sums = Join-Path $dl "SHA256SUMS"
# NOTE: no try/catch here. Under $ErrorActionPreference = "Stop", a try/catch
# around Invoke-WebRequest -OutFile caused Windows PowerShell 5.1 to skip every
# subsequent statement through the next block boundary (observed on Server 2025
# CI, PR #17): $expected/$bin/$actual were never assigned, no catch fired, no
# error printed. -ErrorAction Stop makes IWR throw a terminating error on
# failure (clean exit with its native message) without the skip side effect.
Invoke-WebRequest -Uri "$BASE/releases/download/$vtag/SHA256SUMS" -UseBasicParsing -OutFile $sums -TimeoutSec 60 -ErrorAction Stop
if ($env:GRN_INSTALL_DEBUG) { [Console]::Error.WriteLine("DBG L62 after SHA256SUMS dl: sums=[$sums]") }
# Avoid `Select-Object -First 1` under Stop: it throws a terminating
# StopUpstreamCommandsException in PS 5.1. Index the array instead.
$line = @(Get-Content $sums | Select-String -Pattern "grn-$platform-$vtag.exe`$")[0]
if (-not $line) {
  [Console]::Error.WriteLine("install.ps1: No checksum line for $platform in SHA256SUMS — the release may not ship this platform."); exit 1
}
$expected = ($line.ToString() -split '\s+')[0].ToLower()

# --- Step 5: download binary ---
$bin = Join-Path $dl "grn.exe"
if ($env:GRN_INSTALL_DEBUG) { [Console]::Error.WriteLine("DBG L70 after set: bin=[$bin] dl=[$dl]") }
Invoke-WebRequest -Uri "$BASE/releases/download/$vtag/grn-$platform-$vtag.exe" -UseBasicParsing -OutFile $bin -TimeoutSec 60 -ErrorAction Stop

# --- Step 6: verify ---
$actual = (Get-FileHash -Path $bin -Algorithm SHA256).Hash.ToLower()
if ($env:GRN_INSTALL_DEBUG) { [Console]::Error.WriteLine("DBG L76 after hash: bin=[$bin] actual=[$actual] expected=[$expected]") }
if ($actual -ne $expected) {
  Remove-Item -Force $bin -ErrorAction SilentlyContinue
  [Console]::Error.WriteLine("install.ps1: Checksum mismatch (expected $expected, got $actual) — the download may be corrupt or tampered."); exit 1
}

# --- Step 7: install + PATH ---
$installDir = Join-Path $env:LOCALAPPDATA "greennode\bin"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
$dest = Join-Path $installDir "grn.exe"
if ($env:GRN_INSTALL_DEBUG) { [Console]::Error.WriteLine("DBG L86 before move: bin=[$bin] dl=[$dl] dest=[$dest] installDir=[$installDir] expected=[$expected] actual=[$actual]") }
# Defensive: on Windows PowerShell 5.1 (Server 2025 CI), $bin has been observed
# $null here despite being set at line 70 and consumed by Get-FileHash at line
# 76, with no statement between modifying it. Re-derive from $dl (unchanged
# since line 57) so a transient null doesn't break the install.
if (-not $bin) {
  [Console]::Error.WriteLine("install.ps1: WARNING: `$bin was null at Move-Item; re-deriving from dl=[$dl]")
  $bin = Join-Path $dl "grn.exe"
}
if (-not (Test-Path -LiteralPath $bin)) {
  [Console]::Error.WriteLine("install.ps1: ERROR: binary not found at [$bin] (dl=[$dl])")
  exit 1
}
Move-Item -Path $bin -Destination $dest -Force

# Add to User PATH (idempotent).
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -and ($userPath -notlike "*$installDir*")) {
  [Environment]::SetEnvironmentVariable("PATH", "$userPath;$installDir", "User")
} elseif (-not $userPath) {
  [Environment]::SetEnvironmentVariable("PATH", $installDir, "User")
}
# Refresh current session.
$env:PATH = "$env:PATH;$installDir"

Write-Output "Installed grn $tag -> $dest"
Write-Output "Open a new terminal for PATH to take effect (or run: grn --version)"
