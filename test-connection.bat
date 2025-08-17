@echo off
REM Test script untuk memverifikasi setup

echo ========================================
echo Remote Tunnel - Connection Test
echo ========================================

set RELAY_HOST=sh.adisaputra.online
set RELAY_PORT=443

echo Testing connection to relay server...
echo Host: %RELAY_HOST%
echo Port: %RELAY_PORT%
echo.

REM Test basic connectivity
echo [1/4] Testing basic connectivity...
ping -n 2 %RELAY_HOST% >nul
if errorlevel 1 (
    echo ❌ FAIL: Cannot ping relay server
    echo Check internet connection and server IP
) else (
    echo ✅ PASS: Relay server is reachable
)

echo.
echo [2/4] Testing HTTPS port...
powershell -Command "try { $result = Test-NetConnection -ComputerName '%RELAY_HOST%' -Port %RELAY_PORT% -InformationLevel Quiet; if($result) { Write-Host '✅ PASS: Port %RELAY_PORT% is open' } else { Write-Host '❌ FAIL: Port %RELAY_PORT% is closed or filtered' } } catch { Write-Host '❌ FAIL: Cannot test port %RELAY_PORT%' }"

echo.
echo [3/4] Testing WebSocket endpoint...
REM Test if relay server responds to health check
powershell -Command "try { $response = Invoke-WebRequest -Uri 'https://%RELAY_HOST%/health' -SkipCertificateCheck -TimeoutSec 10; if($response.StatusCode -eq 200) { Write-Host '✅ PASS: Relay server health endpoint responding' } else { Write-Host '❌ FAIL: Relay server health check failed' } } catch { Write-Host '❌ FAIL: Cannot reach relay server health endpoint' }"

echo.
echo [4/4] Testing local agent binary...
if exist "bin\agent.exe" (
    echo ✅ PASS: Agent binary found
) else (
    echo ❌ FAIL: Agent binary not found - run build.bat first
)

echo.
echo ========================================
echo Test Results Summary:
echo ========================================
echo.
echo If all tests pass, you can:
echo 1. On server 103.195.169.32: Run the relay server
echo 2. On your laptop: Run start-agent.bat
echo 3. From remote location: Run start-client.bat
echo.
echo Next steps:
echo - Edit .env.production with your secure token
echo - Run start-agent.bat on this laptop
echo - Configure relay server on 103.195.169.32
echo.
pause
