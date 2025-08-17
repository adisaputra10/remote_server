@echo off
REM Stop relay server safely

echo ========================================
echo Stopping Remote Tunnel Relay Server
echo ========================================

echo Checking for running relay processes...

REM Find and stop relay processes
tasklist /FI "IMAGENAME eq relay.exe" 2>nul | find /I "relay.exe" >nul
if %ERRORLEVEL% EQU 0 (
    echo Found relay processes. Stopping...
    taskkill /F /IM relay.exe >nul 2>&1
    timeout /t 2 /nobreak >nul
    echo ✅ Relay processes stopped
) else (
    echo ℹ️  No relay processes found
)

REM Check if any process is still using port 443 or 8443
echo.
echo Checking ports...

netstat -an | find ":443 " | find "LISTENING" >nul
if %ERRORLEVEL% EQU 0 (
    echo ⚠️  Port 443 still in use
    netstat -an | find ":443 "
) else (
    echo ✅ Port 443 is free
)

netstat -an | find ":8443 " | find "LISTENING" >nul
if %ERRORLEVEL% EQU 0 (
    echo ⚠️  Port 8443 still in use  
    netstat -an | find ":8443 "
) else (
    echo ✅ Port 8443 is free
)

echo.
echo ========================================
echo Relay Server Stopped
echo ========================================
echo.
echo To start relay on port 8443:
echo start-relay.bat
echo.
echo To check ports:
echo netstat -an ^| find ":8443"
echo.
pause
