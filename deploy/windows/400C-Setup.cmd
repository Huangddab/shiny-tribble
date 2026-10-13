@echo off
cd /d "%~dp0"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup.ps1"
if errorlevel 1 (
  echo.
  echo Setup did not complete. Review the error above.
  pause
  exit /b 1
)
echo.
echo Setup step finished. If Windows reboot or Docker first-run is required, rerun this file afterwards.
pause
