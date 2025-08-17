@echo off
REM Monitor connection status for sh.adisaputra.online

echo ========================================
echo Remote Tunnel - Connection Monitor
echo ========================================

set DOMAIN=sh.adisaputra.online
set RELAY_URL=wss://%DOMAIN%/ws

echo Monitoring connection to %DOMAIN%...
echo Press Ctrl+C to stop monitoring
echo.

:MONITOR_LOOP
echo [%date% %time%] Checking connection status...

REM Test basic connectivity
ping -n 1 %DOMAIN% >nul 2>&1
if errorlevel 1 (
    echo ❌ [%time%] CRITICAL: Cannot reach %DOMAIN%
    goto SLEEP_AND_CONTINUE
)

REM Test HTTPS port
powershell -Command "try { $result = Test-NetConnection -ComputerName '%DOMAIN%' -Port 443 -InformationLevel Quiet; if($result) { Write-Host '✅ [%time%] Port 443 is accessible' } else { Write-Host '❌ [%time%] Port 443 is not accessible' } } catch { Write-Host '❌ [%time%] Cannot test port 443' }" 2>nul

REM Test relay health endpoint
powershell -Command "try { $response = Invoke-WebRequest -Uri 'https://%DOMAIN%/health' -TimeoutSec 5; if($response.StatusCode -eq 200) { Write-Host '✅ [%time%] Relay server is healthy' } else { Write-Host '❌ [%time%] Relay server health check failed' } } catch { Write-Host '❌ [%time%] Relay server is not responding' }" 2>nul

REM Check if agent process is running
tasklist /FI "IMAGENAME eq agent.exe" 2>nul | find /I "agent.exe" >nul
if %ERRORLEVEL% EQU 0 (
    echo ✅ [%time%] Agent process is running
) else (
    echo ❌ [%time%] Agent process is not running
)

REM Check if client process is running
tasklist /FI "IMAGENAME eq client.exe" 2>nul | find /I "client.exe" >nul
if %ERRORLEVEL% EQU 0 (
    echo ✅ [%time%] Client process is running
) else (
    echo ⚠️  [%time%] No client process detected
)

:SLEEP_AND_CONTINUE
echo.
timeout /t 30 /nobreak >nul
goto MONITOR_LOOP
