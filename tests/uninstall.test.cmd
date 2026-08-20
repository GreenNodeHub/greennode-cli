@echo off
REM Hermetic smoke test for scripts/uninstall.cmd.
REM Creates the install state in a temp LOCALAPPDATA + User PATH, runs the
REM uninstaller, asserts the binary dir + PATH entry are gone. No HTTP server
REM needed: uninstall is pure local file ops.
setlocal enableextensions enabledelayedexpansion
set "EXITCODE=0"

set "REPO_ROOT=%~dp0.."
pushd "%REPO_ROOT%"
set "REPO_ROOT=%CD%"
popd
set "SCRIPT=%REPO_ROOT%\scripts\uninstall.cmd"
if not exist "%SCRIPT%" ( echo FAIL: uninstall.cmd not found at %SCRIPT% & set "EXITCODE=1" & goto :cleanup )

REM Isolate LOCALAPPDATA + USERPROFILE.
set "ORIG_LA=%LOCALAPPDATA%"
set "ORIG_UP=%USERPROFILE%"
set "TMPDIR=%TEMP%\grn-uninstall-test-%RANDOM%"
mkdir "%TMPDIR%" 2>nul
set "LOCALAPPDATA=%TMPDIR%\localappdata"
set "USERPROFILE=%TMPDIR%\userprofile"
mkdir "%LOCALAPPDATA%" 2>nul
mkdir "%USERPROFILE%" 2>nul

REM Capture User PATH to restore in :cleanup.
set "ORIG_PATH="
for /f "tokens=2,*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul ^| findstr /i "PATH"') do set "ORIG_PATH=%%B"

REM --- Set up install state ---
set "BIN_DIR=%LOCALAPPDATA%\greennode\bin"
mkdir "%BIN_DIR%" 2>nul
echo fake > "%BIN_DIR%\grn.exe"
REM User PATH: sentinel + grn bin dir. Uninstaller must drop only the grn segment.
reg add "HKCU\Environment" /v PATH /t REG_EXPAND_SZ /d "C:\sentinel-path;%BIN_DIR%" /f >nul

REM --- Run uninstaller ---
call "%SCRIPT%"
if errorlevel 1 ( echo FAIL: uninstaller exited errorlevel & set "EXITCODE=1" & goto :cleanup )

REM --- Assertions ---
if exist "%LOCALAPPDATA%\greennode" ( echo FAIL: install dir still exists & set "EXITCODE=1" & goto :cleanup )
set "USERPATH="
for /f "tokens=2,*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul ^| findstr /i "PATH"') do set "USERPATH=%%B"
echo !USERPATH!| findstr /i /c:"%BIN_DIR%" >nul 2>&1
if not errorlevel 1 ( echo FAIL: bin dir still in User PATH & set "EXITCODE=1" & goto :cleanup )
echo !USERPATH!| findstr /i /c:"C:\sentinel-path" >nul 2>&1
if errorlevel 1 ( echo FAIL: sentinel PATH segment was removed & set "EXITCODE=1" & goto :cleanup )

REM --- --purge ---
mkdir "%USERPROFILE%\.greennode" 2>nul
echo creds > "%USERPROFILE%\.greennode\credentials"
call "%SCRIPT%" --purge
if errorlevel 1 ( echo FAIL: uninstaller --purge exited errorlevel & set "EXITCODE=1" & goto :cleanup )
if exist "%USERPROFILE%\.greennode" ( echo FAIL: --purge did not remove config dir & set "EXITCODE=1" & goto :cleanup )

echo PASS: uninstall.cmd removes binary dir + User PATH entry (--purge works)

:cleanup
if defined ORIG_PATH (
  reg add "HKCU\Environment" /v PATH /t REG_EXPAND_SZ /d "!ORIG_PATH!" /f >nul
) else (
  reg delete "HKCU\Environment" /v PATH /f >nul 2>&1
)
rd /s /q "%TMPDIR%" 2>nul 2>&1
set "LOCALAPPDATA=%ORIG_LA%"
set "USERPROFILE=%ORIG_UP%"
endlocal & exit /b %EXITCODE%
