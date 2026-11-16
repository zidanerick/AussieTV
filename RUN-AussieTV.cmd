@echo off
setlocal
cd /d "%~dp0"
title AussieTV Launcher

echo ========================================================
echo  AussieTV 1.5 - Windows Launcher
echo ========================================================
echo.
if not exist "AussieTV.exe" (
  echo ERROR: AussieTV.exe was not found in this folder.
  echo Extract ALL files from the ZIP first.
  echo.
  pause
  exit /b 1
)

echo If no browser opens automatically, try:
echo    http://127.0.0.1:45781

echo The program will print its actual URL if port 45781 is taken.
echo.
echo Starting application...
echo.
"%~dp0AussieTV.exe"
echo.
echo AussieTV exited with code %errorlevel%.
echo If it failed, check %%LOCALAPPDATA%%\AussieTV\startup.log
echo.
pause
