@echo off
cd /d "%~dp0"
title AussieTV Startup Diagnostics
set AUSSIETV_DIAGNOSTICS=1
set AUSSIETV_NO_TRAY=1
"%~dp0AussieTV-Diagnostics.exe"
echo.
echo Press any key to close this diagnostic window.
pause >nul
