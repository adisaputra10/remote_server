@echo off
REM Check relay server status and ports

echo ========================================
echo Remote Tunnel Relay Status Check
echo ========================================

echo [1] Process Status:
tasklist /FI "IMAGENAME eq relay.exe" 2>nul | find /I "relay.exe" >nul
if %ERRORLEVEL% EQU 0 (
    echo ✅ Relay processes running:
    tasklist /FI "IMAGENAME eq relay.exe"
) else (
    echo ❌ No relay processes found
)

echo.
echo [2] Port Status:

REM Check port 443
netstat -an | find ":443" | find "LISTENING" >nul
if %ERRORLEVEL% EQU 0 (
    echo ⚠️  Port 443 is in use:
    netstat -an | find ":443" | find "LISTENING"
) else (
    echo ✅ Port 443 is free
)

REM Check port 8443
netstat -an | find ":8443" | find "LISTENING" >nul
if %ERRORLEVEL% EQU 0 (
    echo ✅ Port 8443 is in use:
    netstat -an | find ":8443" | find "LISTENING"
) else (
    echo ❌ Port 8443 is free
)

echo.
echo [3] Configuration Check:
if exist ".env.production" (
    echo ✅ .env.production exists
    findstr "RELAY_ADDR" .env.production 2>nul
    findstr "RELAY_PORT" .env.production 2>nul
    findstr "TUNNEL_TOKEN" .env.production 2>nul | echo Token configured
) else (
    echo ❌ .env.production not found
)

echo.
echo [4] Binary Check:
if exist "bin\relay.exe" (
    echo ✅ Relay binary exists
    dir "bin\relay.exe" | find "relay.exe"
) else (
    echo ❌ Relay binary not found
)

echo.
echo [5] Certificate Check:
if exist "certs\server.crt" (
    if exist "certs\server.key" (
        echo ✅ Self-signed certificates exist:
        echo    Certificate: certs\server.crt
        echo    Private Key: certs\server.key
    ) else (
        echo ⚠️  Certificate exists but private key missing
    )
) else (
    echo ❌ Self-signed certificates not found
    echo    Run: generate-certs.bat
)

echo.
echo [6] Connectivity Test:
where curl >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo Testing health endpoint...
    curl -k -s "https://sh.adisaputra.online:8443/health" --connect-timeout 5 >nul 2>&1
    if %ERRORLEVEL% EQU 0 (
        echo ✅ Health endpoint responds
    ) else (
        echo ❌ Health endpoint failed
    )
) else (
    echo ⚠️  curl not available for testing
)

echo.
echo ========================================
echo Status Summary
echo ========================================

tasklist /FI "IMAGENAME eq relay.exe" 2>nul | find /I "relay.exe" >nul
set RELAY_RUNNING=%ERRORLEVEL%

netstat -an | find ":8443" | find "LISTENING" >nul
set PORT_8443=%ERRORLEVEL%

netstat -an | find ":443" | find "LISTENING" >nul
set PORT_443=%ERRORLEVEL%

if %RELAY_RUNNING% EQU 0 (
    if %PORT_8443% EQU 0 (
        echo 🟢 Relay server is running correctly on port 8443
    ) else if %PORT_443% EQU 0 (
        echo 🟡 Relay server is running but on port 443 (should be 8443)
        echo    Run: restart-relay.bat
    ) else (
        echo 🟡 Relay process running but port unclear
    )
) else (
    if %PORT_8443% EQU 0 (
        echo 🟡 Port 8443 in use but no relay process found
    ) else (
        echo 🔴 Relay server is not running
        echo    Run: start-relay.bat
    )
)

echo.
echo Commands:
echo - Start: start-relay.bat
echo - Stop:  stop-relay.bat
echo - Restart: restart-relay.bat
echo - Monitor: monitor-connection.bat
echo.
pause
