# GreenNode CLI installer (Windows PowerShell).
# Usage: irm https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.ps1 | iex
param(
  # Allow pinning a version instead of resolving latest (also used by tests).
  [string]$Tag
)
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$ProgressPreference = 'SilentlyContinue'

$BASE = $env:GRN_INSTALL_BASE_URL
if (-not $BASE) { $BASE = "https://github.com/GreenNodeHub/greennode-cli" }
if (-not $Tag) { $Tag = $env:GRN_INSTALL_TAG }

# --- Step 1: 64-bit guard ---
if (-not [Environment]::Is64BitProcess) {
  Write-Error "GreenNode CLI requires 64-bit Windows."; exit 1
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
  Write-Error "Could not resolve a version tag from $BASE/releases/latest (got '$vtag'). Set -Tag or GRN_INSTALL_TAG."; exit 1
}
$tag = $vtag.TrimStart('v')

# --- Step 4: fetch SHA256SUMS + expected hash ---
$dl = Join-Path $env:TEMP ("grn-install-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $dl | Out-Null
$sums = Join-Path $dl "SHA256SUMS"
try {
  Invoke-WebRequest -Uri "$BASE/releases/download/$vtag/SHA256SUMS" -UseBasicParsing -OutFile $sums
} catch { Write-Error "Failed to download SHA256SUMS: $_"; exit 1 }
$line = Get-Content $sums | Select-String -Pattern "grn-$platform-$vtag.exe`$" | Select-Object -First 1
if (-not $line) {
  Write-Error "No checksum line for $platform in SHA256SUMS — the release may not ship this platform."; exit 1
}
$expected = ($line.ToString() -split '\s+')[0].ToLower()

# --- Step 5: download binary ---
$bin = Join-Path $dl "grn.exe"
try {
  Invoke-WebRequest -Uri "$BASE/releases/download/$vtag/grn-$platform-$vtag.exe" -UseBasicParsing -OutFile $bin
} catch { Write-Error "Failed to download binary: $_"; exit 1 }

# --- Step 6: verify ---
$actual = (Get-FileHash -Path $bin -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $expected) {
  Remove-Item -Force $bin -ErrorAction SilentlyContinue
  Write-Error "Checksum mismatch (expected $expected, got $actual) — the download may be corrupt or tampered."; exit 1
}

# --- Step 7: install + PATH ---
$installDir = Join-Path $env:LOCALAPPDATA "greennode\bin"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
$dest = Join-Path $installDir "grn.exe"
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
