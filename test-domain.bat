@echo off
REM Test connection specifically for sh.adisaputra.online

echo ========================================
echo Remote Tunnel - Domain Connection Test
echo ========================================

set DOMAIN=sh.adisaputra.online
set RELAY_PORT=443

echo Testing connection to relay server...
echo Domain: %DOMAIN%
echo Port: %RELAY_PORT%
echo.

echo [1/5] DNS Resolution Test...
nslookup %DOMAIN% >nul 2>&1
if errorlevel 1 (
    echo ❌ FAIL: Cannot resolve domain %DOMAIN%
    echo Check DNS configuration
) else (
    echo ✅ PASS: Domain %DOMAIN% resolves correctly
)

echo.
echo [2/5] Basic Connectivity Test...
ping -n 2 %DOMAIN% >nul
if errorlevel 1 (
    echo ❌ FAIL: Cannot ping relay server
    echo Check internet connection and server status
) else (
    echo ✅ PASS: Relay server is reachable
)

echo.
echo [3/5] HTTPS Port Test...
powershell -Command "try { $result = Test-NetConnection -ComputerName '%DOMAIN%' -Port %RELAY_PORT% -InformationLevel Quiet; if($result) { Write-Host '✅ PASS: Port %RELAY_PORT% is open' } else { Write-Host '❌ FAIL: Port %RELAY_PORT% is closed or filtered' } } catch { Write-Host '❌ FAIL: Cannot test port %RELAY_PORT%' }"

echo.
echo [4/5] TLS Certificate Test...
powershell -Command "try { $request = [System.Net.WebRequest]::Create('https://%DOMAIN%/'); $request.Timeout = 10000; $response = $request.GetResponse(); Write-Host '✅ PASS: TLS connection successful'; $response.Close() } catch { Write-Host '❌ FAIL: TLS connection failed -' $_.Exception.Message }"

echo.
echo [5/5] Relay Server Health Test...
powershell -Command "try { $response = Invoke-WebRequest -Uri 'https://%DOMAIN%/health' -TimeoutSec 10; if($response.StatusCode -eq 200) { Write-Host '✅ PASS: Relay server health endpoint responding' } else { Write-Host '❌ FAIL: Relay server health check failed' } } catch { Write-Host '❌ FAIL: Cannot reach relay server health endpoint -' $_.Exception.Message }"

echo.
echo [Bonus] Agent Binary Check...
if exist "bin\agent.exe" (
    echo ✅ PASS: Agent binary found
) else (
    echo ❌ FAIL: Agent binary not found - run build.bat first
)

echo.
echo ========================================
echo Connection Test Results Summary
echo ========================================
echo.
echo If all tests pass, you can proceed with:
echo.
echo 1. Generate secure token:
echo    generate-token.bat
echo.
echo 2. Start agent on this laptop:
echo    start-agent.bat
echo.
echo 3. From remote machine, create client connection:
echo    bin\client.exe -L :2222 -relay-url wss://%DOMAIN%/ws/client -agent laptop-agent -target 127.0.0.1:22 -token YOUR_TOKEN
echo.
echo 4. Test SSH connection:
echo    ssh -p 2222 user@localhost
echo.
echo Troubleshooting:
echo - If DNS fails: Check domain configuration
echo - If ping fails: Check internet/firewall
echo - If port fails: Check relay server status
echo - If TLS fails: Check certificate configuration
echo - If health fails: Check relay server logs
echo.
pause
