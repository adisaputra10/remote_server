@echo off
REM Cleanup script for remote tunnel

echo ========================================
echo Remote Tunnel - Cleanup
echo ========================================

echo Stopping all tunnel processes...

REM Kill agent processes
tasklist /FI "IMAGENAME eq agent.exe" 2>nul | find /I "agent.exe" >nul
if %ERRORLEVEL% EQU 0 (
    echo Stopping agent processes...
    taskkill /F /IM agent.exe >nul 2>&1
    echo ✅ Agent processes stopped
) else (
    echo ℹ️  No agent processes running
)

REM Kill client processes
tasklist /FI "IMAGENAME eq client.exe" 2>nul | find /I "client.exe" >nul
if %ERRORLEVEL% EQU 0 (
    echo Stopping client processes...
    taskkill /F /IM client.exe >nul 2>&1
    echo ✅ Client processes stopped
) else (
    echo ℹ️  No client processes running
)

REM Kill relay processes
tasklist /FI "IMAGENAME eq relay.exe" 2>nul | find /I "relay.exe" >nul
if %ERRORLEVEL% EQU 0 (
    echo Stopping relay processes...
    taskkill /F /IM relay.exe >nul 2>&1
    echo ✅ Relay processes stopped
) else (
    echo ℹ️  No relay processes running
)

echo.
echo Cleaning up log files...

REM Clean log files
if exist "*.log" (
    del /Q *.log >nul 2>&1
    echo ✅ Log files cleaned
) else (
    echo ℹ️  No log files to clean
)

if exist "logs\*.log" (
    del /Q logs\*.log >nul 2>&1
    echo ✅ Logs directory cleaned
)

REM Clean temp files
if exist "tmp" (
    rmdir /S /Q tmp >nul 2>&1
    echo ✅ Temp directory cleaned
)

echo.
echo Cleaning up build artifacts...

REM Clean Go build cache
echo Cleaning Go build cache...
go clean -cache >nul 2>&1
go clean -modcache >nul 2>&1
echo ✅ Go cache cleaned

echo.
echo ========================================
echo Cleanup Complete!
echo ========================================
echo.
echo What was cleaned:
echo - All running tunnel processes (agent, client, relay)
echo - Log files (*.log, logs/*.log)
echo - Temporary files and directories
echo - Go build cache
echo.
echo What was preserved:
echo - Configuration files (.env.production, config/*.yaml)
echo - Certificates (certs/*)
echo - Built binaries (bin/*)
echo - Source code
echo.
echo To start fresh:
echo 1. Run build.bat to rebuild binaries
echo 2. Run setup-domain.bat for domain configuration
echo 3. Run start-agent.bat to begin tunneling
echo.
pause
