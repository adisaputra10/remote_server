@echo off
REM Quick setup script for sh.adisaputra.online deployment

echo ========================================
echo Remote Tunnel - Quick Setup for Domain
echo ========================================

set DOMAIN=sh.adisaputra.online

echo Setting up remote tunnel for %DOMAIN%...
echo.

echo [1/6] Building binaries...
call build.bat
if errorlevel 1 (
    echo ❌ Build failed!
    pause
    exit /b 1
)

echo.
echo [2/6] Testing domain connectivity...
call test-domain.bat

echo.
echo [3/6] Generating secure token...
call generate-token.bat

echo.
echo [4/6] Configuring environment for %DOMAIN%...
if not exist ".env.production" (
    echo Creating .env.production...
    echo RELAY_HOST=%DOMAIN%> .env.production
    echo RELAY_PORT=8443>> .env.production
    echo RELAY_URL=wss://%DOMAIN%:8443/ws>> .env.production
    echo TLS_ENABLED=true>> .env.production
    echo CERT_FILE=certs/server.crt>> .env.production
    echo KEY_FILE=certs/server.key>> .env.production
    echo TOKEN=your-secure-token-here>> .env.production
    echo LOG_LEVEL=info>> .env.production
    echo.
    echo ⚠️  IMPORTANT: Edit .env.production and set your secure token!
) else (
    echo ✅ .env.production already exists
)

echo.
echo [5/6] Setting up certificates directory...
if not exist "certs" mkdir certs

if not exist "certs\server.crt" (
    echo Generating self-signed certificates...
    if exist "generate-certs.bat" (
        call generate-certs.bat
    ) else (
        echo Manual certificate generation requires OpenSSL
        echo Please run: generate-certs.bat
        echo Or install OpenSSL and run this script again
    )
    echo ✅ Self-signed certificates setup initiated
) else (
    echo ✅ Certificates already exist
)
echo ✅ Certificates directory ready

echo.
echo [6/6] Creating agent configuration...
echo Creating config/agent.yaml...
if not exist "config" mkdir config

(
echo # Agent Configuration for %DOMAIN%
echo agent_id: "laptop-agent"
echo relay_url: "wss://%DOMAIN%:8443/ws/agent"
echo token: "your-secure-token-here"
echo log_level: "info"
echo services:
echo   - name: "ssh"
echo     target: "127.0.0.1:22"
echo   - name: "rdp"
echo     target: "127.0.0.1:3389"
echo   - name: "web"
echo     target: "127.0.0.1:8080"
) > config\agent.yaml

echo ✅ Agent configuration created

echo.
echo ========================================
echo Setup Complete!
echo ========================================
echo.
echo Next steps:
echo.
echo 1. Edit .env.production and set your secure token
echo 2. Edit config\agent.yaml and set the same token
echo 3. Deploy relay to %DOMAIN%:
echo    deploy\deploy-domain.sh (on server)
echo 4. Start agent on this laptop:
echo    start-agent.bat
echo 5. Connect from remote machine:
echo    bin\client.exe -L :2222 -relay-url wss://%DOMAIN%:8443/ws/client -agent laptop-agent -target 127.0.0.1:22 -token YOUR_TOKEN
echo.
echo Security Notes:
echo - Use a strong, unique token (minimum 32 characters)
echo - Configure TLS certificates properly on the relay server
echo - Restrict agent services to only what you need
echo - Monitor logs for suspicious activity
echo.
echo For detailed instructions, see:
echo - QUICKSTART.md
echo - DEPLOYMENT.md
echo.
pause
