@echo off
cd /d "%~dp0"
start "" "opims.exe"
timeout /t 2 /nobreak >nul
start http://localhost:8080
echo OPIMS started. Close this window or press Ctrl+C in the OPIMS window to stop.
