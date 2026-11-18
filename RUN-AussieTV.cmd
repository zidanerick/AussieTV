@echo off
setlocal
cd /d "%~dp0"
if not exist "AussieTV.exe" (
  echo AussieTV.exe is missing. Extract the complete ZIP first.
  pause
  exit /b 1
)
rem Compatibility launcher: the Windows GUI application does not need a console.
start "" "%~dp0AussieTV.exe"
exit /b
