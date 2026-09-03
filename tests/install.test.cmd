@echo off
setlocal enableextensions enabledelayedexpansion
REM Hermetic smoke test for scripts\install.cmd.
REM Stands up a fake release and runs the installer against it (GRN_INSTALL_TAG skips tag-resolve).

set "REPO_ROOT=%~dp0.."
set "TMP=%TEMP%\grn-cmd-test-%RANDOM%"
mkdir "%TMP%\srv\releases\download\v9.9.9" 2>nul

REM Fake grn.exe (a batch stub that prints a version).
>"%TMP%\srv\releases\download\v9.9.9\grn-windows-amd64-v9.9.9.exe" echo @echo off
>>"%TMP%\srv\releases\download\v9.9.9\grn-windows-amd64-v9.9.9.exe" echo echo grn-cli/9.9.9 fake

REM Compute its SHA256 via certutil and write SHA256SUMS.
set "FAKE=%TMP%\srv\releases\download\v9.9.9\grn-windows-amd64-v9.9.9.exe"
certutil -hashfile "%FAKE%" SHA256 > "%TMP%\hashout.txt"
REM Line 2 of certutil output is the hash.
set "HASH="
set /a _n=0
for /f "delims=" %%L in (%TMP%\hashout.txt) do (
  set /a _n+=1
  if !_n! equ 2 set "HASH=%%L"
)
>"%TMP%\srv\releases\download\v9.9.9\SHA256SUMS" echo !HASH!  grn-windows-amd64-v9.9.9.exe

REM Start a static HTTP server (no 302 needed; GRN_INSTALL_TAG skips resolve).
REM Use a port outside the ps1 test's 18081-18085 range to avoid any lingering
REM bind from the prior step. Poll for readiness with curl instead of `timeout /t`,
REM which fails in non-interactive CI ("ERROR: Input redirection is not supported").
where python >nul 2>&1
if errorlevel 1 ( echo FAIL: python not on PATH & exit /b 1 )
set "PORT=18091"
pushd "%TMP%\srv"
start /b "" python -m http.server !PORT!
popd
set /a _tries=0
:wait_server
curl.exe -fsS -o NUL "http://127.0.0.1:!PORT!/releases/download/v9.9.9/SHA256SUMS" >nul 2>&1
if not errorlevel 1 goto :server_ready
set /a _tries+=1
if !_tries! lss 50 goto :wait_server
echo FAIL: test HTTP server did not start on port !PORT!
exit /b 1
:server_ready

set "GRN_INSTALL_BASE_URL=http://127.0.0.1:!PORT!"
set "GRN_INSTALL_TAG=v9.9.9"
set "LOCALAPPDATA=%TMP%\localappdata"
mkdir "%LOCALAPPDATA%" 2>nul

call "%REPO_ROOT%\scripts\install.cmd"
set "EXIT=%ERRORLEVEL%"

REM Stop the server (best-effort).
for /f "tokens=5" %%P in ('netstat -ano ^| findstr :!PORT! ^| findstr LISTENING') do taskkill /pid %%P /f >nul 2>&1

if not "%EXIT%"=="0" ( echo FAIL: install.cmd exited %EXIT% & exit /b 1 )
if not exist "%LOCALAPPDATA%\greennode\bin\grn.exe" ( echo FAIL: grn.exe not installed & exit /b 1 )
echo PASS: install.cmd installed grn.exe to %LOCALAPPDATA%\greennode\bin\grn.exe
endlocal
