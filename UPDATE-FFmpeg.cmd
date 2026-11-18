@echo off
setlocal
set "FFMPEG_DIR=%~dp0ffmpeg"
if exist "%~dp0ffmpeg-location.txt" (
  for /f "usebackq delims=" %%F in ("%~dp0ffmpeg-location.txt") do set "FFMPEG_DIR=%%~dpF"
)
echo Updating FFmpeg in "%FFMPEG_DIR%"...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0tools\ffmpeg\fetch-windows.ps1" -Destination "%FFMPEG_DIR%" -AppRoot "%~dp0"
if errorlevel 1 (
  echo FFmpeg update failed. The existing binary remains available if installed.
  pause
  exit /b 1
)
echo FFmpeg update complete.
pause
