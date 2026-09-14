# Hermetic smoke test for scripts/install.ps1 (Windows PowerShell).
# Requires: PowerShell 5.1+ (Windows). Plain asserts, no Pester.
#
# Stands up a fake release behind a local HttpListener, points the installer at
# it via GRN_INSTALL_BASE_URL + GRN_INSTALL_TAG (GRN_INSTALL_TAG makes the
# installer SKIP the /releases/latest 302 tag-resolve — the script's production
# tag-resolve path is exercised only against real GitHub, never here), redirects
# LOCALAPPDATA + TEMP to a temp dir, runs the installer twice, and asserts a
# correct, checksum-verified, idempotent install.
#
# Run on windows-latest:  powershell -ExecutionPolicy Bypass -File tests\install.test.ps1
$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest
$ProgressPreference = 'SilentlyContinue'

# Count how many ';'-delimited PATH segments exactly equal $segment (case-insensitive).
# Returns 0 when $value is $null/empty (Set-StrictMode-safe).
function Get-PathSegmentCount($value, $segment) {
    if (-not $value) { return 0 }
    $n = 0
    foreach ($s in $value -split ';') { if ($s.Trim() -ieq $segment) { $n++ } }
    return $n
}

$RepoRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$Script   = Join-Path $RepoRoot "scripts\install.ps1"
if (-not (Test-Path -LiteralPath $Script)) { Write-Error "install.ps1 not found at $Script"; exit 1 }

# --- Capture env we will override, to restore in finally (hermetic on dev box) -
$origLocalAppData = $env:LOCALAPPDATA
$origTemp         = $env:TEMP
$origBaseUrl      = $env:GRN_INSTALL_BASE_URL
$origTag          = $env:GRN_INSTALL_TAG
$origDebug        = $env:GRN_INSTALL_DEBUG
$origUserPath     = [Environment]::GetEnvironmentVariable("PATH", "User")

$Tmp = Join-Path $env:TEMP ("grn-install-test-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $Tmp | Out-Null

# --- Fake release tree (served root = $Srv; /releases/download/v9.9.9/...) ----
$Srv   = Join-Path $Tmp "srv"
$DlDir = Join-Path $Srv "releases\download\v9.9.9"
New-Item -ItemType Directory -Force -Path $DlDir | Out-Null

# $env:PROCESSOR_ARCHITECTURE is read-only in PowerShell, so we cannot stub it;
# the fake file MUST match the host's real arch. On windows-latest (amd64) this
# is windows-amd64. ARM64 Windows is a follow-up (adjust $FakeName there).
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { $platform = "windows-arm64" }
else { $platform = "windows-amd64" }
$FakeName = "grn-$platform-v9.9.9.exe"
$FakeExe  = Join-Path $DlDir $FakeName
# A tiny batch stub is enough: the installer only hashes + moves the file; it
# never executes it.
Set-Content -Path $FakeExe -Value "@echo off`r`necho grn-cli/9.9.9 fake" -Encoding ASCII
$Hash = (Get-FileHash -Path $FakeExe -Algorithm SHA256).Hash.ToLower()
# GNU-format SHA256SUMS: "<hash>  <filename>" (two spaces) — matches Task 1.
Set-Content -Path (Join-Path $DlDir "SHA256SUMS") -Value ("{0}  {1}" -f $Hash, $FakeName) -Encoding ASCII

# --- Local HttpListener server: serves files from $Srv by URL path -----------
# (HttpListener is PowerShell's simplest static server -- no python equivalent.)
# About a request to /releases/latest: GRN_INSTALL_TAG skips it, so we do NOT
# emulate the 302 here. Any unmapped path -> 404.
$serve = {
    param($root, $port)
    $L = New-Object System.Net.HttpListener
    $L.Prefixes.Add("http://127.0.0.1:$port/")
    $L.Start()
    while ($L.IsListening) {
        $ctx = $L.GetContext()
        try {
            $rel  = $ctx.Request.Url.AbsolutePath.TrimStart("/").Replace("/", "\")
            $file = Join-Path $root $rel
            if (Test-Path -LiteralPath $file -PathType Leaf) {
                $bytes = [System.IO.File]::ReadAllBytes($file)
                $ctx.Response.ContentLength64 = $bytes.Length
                $ctx.Response.OutputStream.Write($bytes, 0, $bytes.Length)
            } else {
                $ctx.Response.StatusCode = 404
            }
        } catch {
            $ctx.Response.StatusCode = 500
        } finally {
            $ctx.Response.Close()
        }
    }
}

# Bind a fixed high port; retry a few candidates in case one is taken. Probe a
# known file until the server answers (Start-Job startup is not instantaneous).
$Port = 0
$Job  = $null
foreach ($candidate in 18081, 18082, 18083, 18084, 18085) {
    $jobTry = Start-Job -ScriptBlock $serve -ArgumentList $Srv, $candidate
    $ready  = $false
    $probe  = "http://127.0.0.1:$candidate/releases/download/v9.9.9/SHA256SUMS"
    for ($i = 0; $i -lt 25; $i++) {
        if ($jobTry.State -ne "Running" -and $jobTry.State -ne "NotStarted") { break }
        try {
            $r = Invoke-WebRequest -Uri $probe -UseBasicParsing -TimeoutSec 2
            if ($r.StatusCode -eq 200) { $ready = $true; break }
        } catch { }
        Start-Sleep -Milliseconds 200
    }
    if ($ready) { $Port = $candidate; $Job = $jobTry; break }
    Stop-Job  $jobTry -ErrorAction SilentlyContinue
    Remove-Job $jobTry -ErrorAction SilentlyContinue
}
if ($Port -eq 0 -or -not $Job) { Write-Error "FAIL: test HTTP server did not start on any candidate port"; exit 1 }

# --- Point the installer at the fake release + isolate LOCALAPPDATA/TEMP -----
$env:GRN_INSTALL_BASE_URL = "http://127.0.0.1:$Port"
$env:GRN_INSTALL_TAG      = "v9.9.9"   # skip tag-resolve (/releases/latest is not served)
$env:GRN_INSTALL_DEBUG    = "1"        # enable install.ps1 debug traces on stderr
$env:LOCALAPPDATA         = Join-Path $Tmp "localappdata"
$env:TEMP                 = $Tmp        # so the installer's download temp lands under $Tmp
New-Item -ItemType Directory -Force -Path $env:LOCALAPPDATA | Out-Null

$installDir = Join-Path $env:LOCALAPPDATA "greennode\bin"
$installed  = Join-Path $installDir "grn.exe"

# Use powershell.exe (PS 5.1 — the irm|iex target) when present, else pwsh.
if (Get-Command powershell -ErrorAction SilentlyContinue) { $psExe = "powershell" }
else { $psExe = "pwsh" }

# Robust to a pre-existing installDir in User PATH: expect delta of exactly +1.
$origCount = Get-PathSegmentCount $origUserPath $installDir

try {
    # --- Run 1 ----------------------------------------------------------------
    & $psExe -ExecutionPolicy Bypass -File $Script
    if ($LASTEXITCODE -ne 0) { throw "installer exited $LASTEXITCODE on run 1" }
    if (-not (Test-Path -LiteralPath $installed)) { throw "FAIL: grn.exe not installed at $installed" }

    # The installed file must be the checksum-verified one we served.
    $installedHash = (Get-FileHash -Path $installed -Algorithm SHA256).Hash.ToLower()
    if ($installedHash -ne $Hash) { throw "FAIL: installed hash $installedHash != expected $Hash" }

    $count1 = Get-PathSegmentCount ([Environment]::GetEnvironmentVariable("PATH", "User")) $installDir
    if ($count1 -ne ($origCount + 1)) { throw "FAIL: expected installDir +1 in User PATH after run 1 (orig=$origCount, got $count1)" }

    # --- Run 2 (idempotency: must not append installDir again) ----------------
    & $psExe -ExecutionPolicy Bypass -File $Script
    if ($LASTEXITCODE -ne 0) { throw "installer exited $LASTEXITCODE on run 2" }
    if (-not (Test-Path -LiteralPath $installed)) { throw "FAIL: grn.exe missing after run 2" }

    $count2 = Get-PathSegmentCount ([Environment]::GetEnvironmentVariable("PATH", "User")) $installDir
    if ($count2 -ne ($origCount + 1)) { throw "FAIL: idempotency broken -- installDir delta $count2 != $($origCount + 1) after run 2" }

    Write-Output "PASS: install.ps1 installed grn.exe to $installed (idempotent PATH)"
}
finally {
    # Restore everything we touched (hermetic on a dev Windows box; CI is ephemeral).
    [Environment]::SetEnvironmentVariable("PATH", [string]$origUserPath, "User")
    $env:LOCALAPPDATA         = $origLocalAppData
    $env:TEMP                 = $origTemp
    $env:GRN_INSTALL_BASE_URL = $origBaseUrl
    $env:GRN_INSTALL_TAG      = $origTag
    $env:GRN_INSTALL_DEBUG    = $origDebug
    if ($Job) {
        Stop-Job  $Job -ErrorAction SilentlyContinue
        Remove-Job $Job -ErrorAction SilentlyContinue
    }
    Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}
