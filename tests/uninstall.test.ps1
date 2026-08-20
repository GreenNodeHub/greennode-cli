# Hermetic test for scripts/uninstall.ps1.
# Creates the install state (binary dir + User PATH entry) in a temp LOCALAPPDATA,
# runs the uninstaller, asserts everything is gone — then idempotency + -Purge.
# No HTTP server needed: uninstall is pure local file ops.
$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest
$ProgressPreference = 'SilentlyContinue'

$RepoRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$Script   = Join-Path $RepoRoot "scripts\uninstall.ps1"
if (-not (Test-Path -LiteralPath $Script)) { Write-Error "uninstall.ps1 not found at $Script"; exit 1 }

# --- Capture env to restore in finally (hermetic on dev box; CI is ephemeral) --
$origLocalAppData = $env:LOCALAPPDATA
$origUserProfile  = $env:USERPROFILE
$origUserPath     = [Environment]::GetEnvironmentVariable("PATH", "User")

$Tmp = Join-Path $env:TEMP ("grn-uninstall-test-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $Tmp | Out-Null

$env:LOCALAPPDATA = Join-Path $Tmp "localappdata"
$env:USERPROFILE  = Join-Path $Tmp "userprofile"
New-Item -ItemType Directory -Force -Path $env:LOCALAPPDATA | Out-Null
New-Item -ItemType Directory -Force -Path $env:USERPROFILE | Out-Null

$installDir = Join-Path $env:LOCALAPPDATA "greennode"
$binDir     = Join-Path $installDir "bin"
$installed  = Join-Path $binDir "grn.exe"

# Use powershell.exe (PS 5.1 — the irm|iex target) when present, else pwsh.
if (Get-Command powershell -ErrorAction SilentlyContinue) { $psExe = "powershell" }
else { $psExe = "pwsh" }

try {
    # --- Set up install state -------------------------------------------------
    New-Item -ItemType Directory -Force -Path $binDir | Out-Null
    Set-Content -Path $installed -Value "fake" -Encoding ASCII
    # User PATH: a sentinel segment + the grn bin dir. The uninstaller must drop
    # only the grn segment, leaving the sentinel intact.
    [Environment]::SetEnvironmentVariable("PATH", "C:\sentinel-path;$binDir", "User")

    # --- Run 1 ----------------------------------------------------------------
    & $psExe -ExecutionPolicy Bypass -File $Script
    if ($LASTEXITCODE -ne 0) { throw "uninstaller exited $LASTEXITCODE on run 1" }
    if (Test-Path -LiteralPath $installDir) { throw "FAIL: install dir still exists" }

    $p1 = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($p1 -notlike "*C:\sentinel-path*") { throw "FAIL: sentinel PATH segment was removed (over-removal)" }
    if ($p1 -like "*$binDir*") { throw "FAIL: binDir still in User PATH after run 1" }

    # --- Run 2 (idempotency) --------------------------------------------------
    & $psExe -ExecutionPolicy Bypass -File $Script
    if ($LASTEXITCODE -ne 0) { throw "uninstaller exited $LASTEXITCODE on run 2" }

    # --- -Purge ---------------------------------------------------------------
    $configDir = Join-Path $env:USERPROFILE ".greennode"
    New-Item -ItemType Directory -Force -Path $configDir | Out-Null
    Set-Content -Path (Join-Path $configDir "credentials") -Value "creds" -Encoding ASCII
    $legacyDir = Join-Path $env:USERPROFILE ".greenode"
    New-Item -ItemType Directory -Force -Path $legacyDir | Out-Null

    & $psExe -ExecutionPolicy Bypass -File $Script -Purge
    if ($LASTEXITCODE -ne 0) { throw "uninstaller exited $LASTEXITCODE on -Purge" }
    if (Test-Path -LiteralPath $configDir) { throw "FAIL: -Purge did not remove ~/.greennode" }
    if (Test-Path -LiteralPath $legacyDir) { throw "FAIL: -Purge did not remove legacy ~/.greenode" }

    Write-Output "PASS: uninstall.ps1 removes binary dir + User PATH entry (idempotent, -Purge works)"
}
finally {
    [Environment]::SetEnvironmentVariable("PATH", [string]$origUserPath, "User")
    $env:LOCALAPPDATA = $origLocalAppData
    $env:USERPROFILE  = $origUserProfile
    Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}
