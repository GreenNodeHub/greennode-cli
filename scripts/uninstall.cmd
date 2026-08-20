@echo off
REM GreenNode CLI uninstaller (Windows pure-CMD) — for environments without PowerShell.
REM Removes what scripts/install.cmd put down: %LOCALAPPDATA%\greennode\ and its PATH entry.
REM Usage: curl -fsSL https://raw.githubusercontent.com/.../scripts/uninstall.cmd -o uninstall.cmd && uninstall.cmd && del uninstall.cmd
REM   uninstall.cmd --purge   (also remove %USERPROFILE%\.greennode config + credentials)
setlocal enableextensions enabledelayedexpansion
set "EXITCODE=0"

set "PURGE=0"
if /i "%~1"=="--purge" set "PURGE=1"

REM --- Step 1: remove binary dir ---
set "INSTALL_DIR=%LOCALAPPDATA%\greennode"
if exist "%INSTALL_DIR%" rd /s /q "%INSTALL_DIR%" 2>nul

REM --- Step 2: remove %INSTALL_DIR%\bin from User PATH (registry read-modify-write) ---
set "BIN_DIR=%INSTALL_DIR%\bin"
set "USERPATH="
for /f "tokens=2,*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul ^| findstr /i "PATH"') do set "USERPATH=%%B"
if not defined USERPATH goto :purge
REM Rebuild PATH excluding BIN_DIR (case-insensitive). Flat label flow (no nested
REM labels inside parens — batch can't call a label from within a parenthesized block).
set "NEWPATH="
set "P=!USERPATH!"
:split
for /f "delims=;" %%X in ("!P!") do set "SEG=%%X"
if not "!SEG!"=="" if /i not "!SEG!"=="!BIN_DIR!" (
  if defined NEWPATH ( set "NEWPATH=!NEWPATH!;!SEG!" ) else ( set "NEWPATH=!SEG!" )
)
set "TAIL=!P:*;=!"
if "!TAIL!"=="!P!" goto :writepath
set "P=!TAIL!"
goto :split
:writepath
if defined NEWPATH (
  reg add "HKCU\Environment" /v PATH /t REG_EXPAND_SZ /d "!NEWPATH!" /f >nul
) else (
  reg delete "HKCU\Environment" /v PATH /f >nul 2>&1
)

:purge
if not "%PURGE%"=="1" goto :done
if exist "%USERPROFILE%\.greennode" rd /s /q "%USERPROFILE%\.greennode" 2>nul
if exist "%USERPROFILE%\.greenode" rd /s /q "%USERPROFILE%\.greenode" 2>nul

:done
echo Uninstalled grn
if "%PURGE%"=="1" echo Also removed %USERPROFILE%\.greennode (config + credentials)
echo Open a new terminal for PATH changes to take effect.
endlocal & exit /b %EXITCODE%
