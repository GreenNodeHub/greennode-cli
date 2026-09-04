@echo off
REM GreenNode CLI installer (Windows pure-CMD) — for environments without PowerShell.
REM Usage: curl -fsSL https://raw.githubusercontent.com/.../scripts/install.cmd -o install.cmd && install.cmd && del install.cmd
setlocal enableextensions enabledelayedexpansion
set "EXITCODE=0"

set "BASE=%GRN_INSTALL_BASE_URL%"
if "%BASE%"=="" set "BASE=https://github.com/GreenNodeHub/greennode-cli"

REM --- Step 1: arch detect + 32-bit reject ---
set "PLATFORM=windows-amd64"
if /i "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "PLATFORM=windows-arm64"
if /i "%PROCESSOR_ARCHITECTURE%"=="x86" goto :err_32bit

REM --- Step 3: resolve tag (GRN_INSTALL_TAG skips) ---
if not "%GRN_INSTALL_TAG%"=="" (
  set "VTAG=%GRN_INSTALL_TAG%"
  goto :tag_done
)
set "REDIRECT="
for /f "delims=" %%i in ('curl.exe -fsSL -o NUL -w "%%{url_effective}" "%BASE%/releases/latest" 2^>^&1') do set "REDIRECT=%%i"
if "%REDIRECT%"=="" goto :err_resolve
REM Extract the part after /tag/ (gives v1.12.0).
set "VTAG=%REDIRECT:*/tag/=%"
:tag_done
REM Validate VTAG looks like vX.Y.Z.
echo !VTAG!| findstr /r "^v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*" >nul || goto :err_resolve
set "TAG=!VTAG:~1!"

REM --- Step 4: fetch SHA256SUMS + expected hash ---
set "TMP=%TEMP%\grn-cmd-%RANDOM%"
mkdir "%TMP%" 2>nul
curl.exe -fsSL "%BASE%/releases/download/!VTAG!/SHA256SUMS" -o "%TMP%\SHA256SUMS" || goto :err_dl_sums
set "EXPECTED="
for /f "tokens=1" %%h in ('findstr /c:"grn-!PLATFORM!-!VTAG!.exe" "%TMP%\SHA256SUMS"') do set "EXPECTED=%%h"
if "%EXPECTED%"=="" goto :err_no_platform

REM --- Step 5: download binary (+ corporate-proxy cert-revocation retry) ---
curl.exe -fsSL "%BASE%/releases/download/!VTAG!/grn-!PLATFORM!-!VTAG!.exe" -o "%TMP%\grn-installer.exe" 2>"%TMP%\dlerr.txt"
if errorlevel 1 (
  findstr /c:"0x80092012" /c:"0x80092013" "%TMP%\dlerr.txt" >nul 2>&1 && (
    curl.exe -fsSL --ssl-revoke-best-effort "%BASE%/releases/download/!VTAG!/grn-!PLATFORM!-!VTAG!.exe" -o "%TMP%\grn-installer.exe" || goto :err_dl_bin
  ) || goto :err_dl_bin
)

REM --- Step 6: verify (certutil; parse line 2) ---
certutil -hashfile "%TMP%\grn-installer.exe" SHA256 > "%TMP%\hashout.txt"
set "ACTUAL="
set /a _n=0
for /f "delims=" %%L in (%TMP%\hashout.txt) do (
  set /a _n+=1
  if !_n! equ 2 set "ACTUAL=%%L"
)
REM certutil emits uppercase hex; lowercase-compare by uppercasing EXPECTED.
if /i not "!ACTUAL!"=="!EXPECTED!" ( del "%TMP%\grn-installer.exe" >nul 2>&1 & goto :err_checksum )

REM --- Step 7: install + PATH (registry; robust vs setx truncation) ---
set "INSTALL_DIR=%LOCALAPPDATA%\greennode\bin"
mkdir "%INSTALL_DIR%" 2>nul
move /y "%TMP%\grn-installer.exe" "%INSTALL_DIR%\grn.exe" >nul || goto :err_move

REM Read HKCU\Environment\PATH, append INSTALL_DIR if absent, write back REG_EXPAND_SZ.
set "USERPATH="
for /f "tokens=2,*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul ^| findstr /i "PATH"') do set "USERPATH=%%B"
echo !USERPATH!| findstr /i /c:"%INSTALL_DIR%" >nul 2>&1
if errorlevel 1 (
  if defined USERPATH ( set "NEWPATH=!USERPATH!;%INSTALL_DIR%" ) else ( set "NEWPATH=%INSTALL_DIR%" )
  reg add "HKCU\Environment" /v PATH /t REG_EXPAND_SZ /d "!NEWPATH!" /f >nul
)
REM Refresh current session.
set "PATH=%PATH%;%INSTALL_DIR%"

echo Installed grn !TAG! -^> %INSTALL_DIR%\grn.exe
echo Open a new terminal for PATH to take effect (or run: grn --version)
goto :cleanup

:err_32bit
echo Error: GreenNode CLI requires 64-bit Windows. & goto :cleanup_err
:err_resolve
echo Error: could not resolve latest version tag from %BASE%/releases/latest. Set GRN_INSTALL_TAG=vX.Y.Z. & goto :cleanup_err
:err_dl_sums
echo Error: failed to download SHA256SUMS for !VTAG!. & goto :cleanup_err
:err_no_platform
echo Error: no checksum for !PLATFORM! in SHA256SUMS -- release may not ship this platform. & goto :cleanup_err
:err_dl_bin
echo Error: failed to download binary. & goto :cleanup_err
:err_checksum
echo Error: checksum mismatch -- download may be corrupt or tampered. & goto :cleanup_err
:err_move
echo Error: could not install grn.exe (is grn currently running? close it and retry). & goto :cleanup_err

:cleanup_err
set "EXITCODE=1"
:cleanup
rd /s /q "%TMP%" >nul 2>&1
endlocal & exit /b %EXITCODE%
