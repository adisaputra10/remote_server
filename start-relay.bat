@echo off
REM Production setup script for relay server
REM To be run on server

echo ========================================
echo Remote Tunnel - Relay Server Setup
echo ========================================
echo Server Domain: sh.adisaputra.online
echo Listening on: Port 8443 (HTTPS/WSS)
echo ========================================

REM Load configuration
if exist .env.production (
    echo Loading production configuration...
    for /f "tokens=1,2 delims==" %%a in (.env.production) do (
        if not "%%a"=="#" (
            set %%a=%%b
        )
    )
    if "%RELAY_ADDR%"=="" (
        set RELAY_ADDR=:8443
        echo Warning: RELAY_ADDR not set, using default :8443
    )
) else (
    echo Warning: .env.production not found, using defaults
    set TUNNEL_TOKEN=change-this-token
    set RELAY_ADDR=:8443
)

echo.
echo Configuration:
echo - Token: %TUNNEL_TOKEN%
echo - Listen Address: %RELAY_ADDR%
echo - Certificate: auto-generated
echo.

REM Check if binaries exist
if not exist "bin\relay.exe" (
    echo Error: relay.exe binary not found. Building...
    call build.bat
    if errorlevel 1 (
        echo Build failed!
        pause
        exit /b 1
    )
)

REM Create certificate directory
if not exist "certs" mkdir certs

REM Generate self-signed certificates if they don't exist
if not exist "certs\server.crt" (
    echo Self-signed certificates not found. Generating...
    if exist "generate-certs.bat" (
        call generate-certs.bat
    ) else (
        echo Manual certificate generation would require OpenSSL...
        echo Please run generate-certs.bat or install OpenSSL
    )
    echo Self-signed certificates generated
)

REM Set certificate paths
if exist "certs\server.crt" (
    set CERT_ARGS=-cert certs\server.crt -key certs\server.key
    echo Using self-signed certificates
) else (
    set CERT_ARGS=
    echo No certificates found - relay will auto-generate basic ones
)

echo.
echo Compression options:
echo [1] No compression support (faster processing)
echo [2] Enable compression support (better bandwidth utilization)
set /p compression_choice="Select compression option (1-2) [1]: "
if "%compression_choice%"=="" set compression_choice=1

if "%compression_choice%"=="2" (
    set COMPRESSION_FLAG=-compress
    echo Selected: Compression support enabled
) else (
    set COMPRESSION_FLAG=
    echo Selected: No compression support
)

echo.
echo Starting relay server...
echo Command: bin\relay.exe -addr %RELAY_ADDR% %CERT_ARGS% -token %TUNNEL_TOKEN% %COMPRESSION_FLAG%
echo.
echo Endpoints:
echo - Agent: wss://sh.adisaputra.online:8443/ws/agent
echo - Client: wss://sh.adisaputra.online:8443/ws/client
echo - Health: https://sh.adisaputra.online:8443/health
echo.
echo Press Ctrl+C to stop
echo ========================================

REM Start relay server
bin\relay.exe -addr %RELAY_ADDR% %CERT_ARGS% -token %TUNNEL_TOKEN% %COMPRESSION_FLAG%

echo.
echo Relay server stopped.
pause
