# GreenNode CLI uninstaller (Windows PowerShell).
# Removes what scripts/install.ps1 put down: the %LOCALAPPDATA%\greennode binary
# dir and its entry in the User PATH.
#
# Usage: irm https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/uninstall.ps1 | iex
#   With -Purge (also remove %USERPROFILE%\.greennode config + credentials):
#     & ([scriptblock]::Create((irm https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/uninstall.ps1))) -Purge
param(
  [switch]$Purge
)
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$ProgressPreference = 'SilentlyContinue'

$installDir = Join-Path $env:LOCALAPPDATA "greennode"

# --- Step 1: remove binary dir ---
if (Test-Path -LiteralPath $installDir) {
  Remove-Item -Recurse -Force -LiteralPath $installDir
}

# --- Step 2: remove from User PATH (idempotent) ---
$binInPath = Join-Path $installDir "bin"
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath) {
  # Drop empty segments + any segment matching binInPath (case-insensitive).
  $segments = $userPath -split ';' | Where-Object { $_ -and ($_.Trim() -ine $binInPath) }
  [Environment]::SetEnvironmentVariable("PATH", ($segments -join ';'), "User")
}

# --- Step 3: -Purge config + credentials ---
if ($Purge) {
  foreach ($name in @(".greennode", ".greenode")) {   # current + legacy pre-rename dir
    $d = Join-Path $env:USERPROFILE $name
    if (Test-Path -LiteralPath $d) { Remove-Item -Recurse -Force -LiteralPath $d }
  }
}

Write-Output "Uninstalled grn"
if ($Purge) { Write-Output "Also removed $env:USERPROFILE\.greennode (config + credentials)" }
Write-Output "Open a new terminal for PATH changes to take effect."
