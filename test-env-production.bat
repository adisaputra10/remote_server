@echo off
REM test-env-production.bat
REM Script untuk test konfigurasi .env.production di Windows

echo ===============================================
echo Testing .env.production Configuration
echo ===============================================

REM Check if .env.production exists
if not exist ".env.production" (
    echo ❌ Error: .env.production file not found!
    echo Please create .env.production file first
    pause
    exit /b 1
)

echo ✅ .env.production file found
echo.

REM Load environment variables from .env.production
echo Loading environment variables...
for /f "usebackq tokens=1,2 delims==" %%a in (`.env.production`) do (
    if not "%%a"=="" if not "%%a:~0,1%%"=="#" (
        set "%%a=%%b"
    )
)

echo ✅ Environment variables loaded
echo.

REM Display configuration
echo Production Configuration Summary:
echo =================================
echo.

echo 🔐 Security:
echo - Token: %TUNNEL_TOKEN:~0,10%...%TUNNEL_TOKEN:~-4% 

echo.
echo 🖥️  Relay Server:
if defined RELAY_HOST (echo - Host: %RELAY_HOST%) else (echo - Host: not set)
if defined RELAY_PORT (echo - Port: %RELAY_PORT%) else (echo - Port: not set)
if defined RELAY_ADDR (echo - Address: %RELAY_ADDR%) else (echo - Address: not set)
if defined RELAY_CERT_FILE (echo - Certificate: %RELAY_CERT_FILE%) else (echo - Certificate: not set)
if defined RELAY_KEY_FILE (echo - Private Key: %RELAY_KEY_FILE%) else (echo - Private Key: not set)
if defined TLS_ENABLED (echo - TLS Enabled: %TLS_ENABLED%) else (echo - TLS Enabled: not set)

echo.
echo 🤖 Agent:
if defined AGENT_ID (echo - ID: %AGENT_ID%) else (echo - ID: not set)
if defined AGENT_RELAY_URL (echo - Relay URL: %AGENT_RELAY_URL%) else (echo - Relay URL: not set)

echo.
echo 🔌 Allowed Services:
if defined AGENT_ALLOW_SSH (echo - SSH: %AGENT_ALLOW_SSH%) else (echo - SSH: not configured)
if defined AGENT_ALLOW_HTTP (echo - HTTP: %AGENT_ALLOW_HTTP%) else (echo - HTTP: not configured)
if defined AGENT_ALLOW_HTTPS (echo - HTTPS: %AGENT_ALLOW_HTTPS%) else (echo - HTTPS: not configured)
if defined AGENT_ALLOW_WEB (echo - Web Dev: %AGENT_ALLOW_WEB%) else (echo - Web Dev: not configured)
if defined AGENT_ALLOW_DEV (echo - Dev Server: %AGENT_ALLOW_DEV%) else (echo - Dev Server: not configured)
if defined AGENT_ALLOW_POSTGRES (echo - PostgreSQL: %AGENT_ALLOW_POSTGRES%) else (echo - PostgreSQL: not configured)
if defined AGENT_ALLOW_MYSQL (echo - MySQL: %AGENT_ALLOW_MYSQL%) else (echo - MySQL: not configured)
if defined AGENT_ALLOW_REDIS (echo - Redis: %AGENT_ALLOW_REDIS%) else (echo - Redis: not configured)

echo.
echo 📱 Client:
if defined CLIENT_RELAY_URL (echo - Relay URL: %CLIENT_RELAY_URL%) else (echo - Relay URL: not set)
if defined CLIENT_SSH_PORT (echo - SSH Port: %CLIENT_SSH_PORT%) else (echo - SSH Port: not set)
if defined CLIENT_WEB_PORT (echo - Web Port: %CLIENT_WEB_PORT%) else (echo - Web Port: not set)
if defined CLIENT_DB_PORT (echo - DB Port: %CLIENT_DB_PORT%) else (echo - DB Port: not set)

echo.
echo 🔍 Validation:
echo ==============

set ERRORS=0

REM Validate required variables
if not defined TUNNEL_TOKEN (
    echo ❌ TUNNEL_TOKEN is required
    set /a ERRORS+=1
)

if not defined RELAY_HOST (
    echo ❌ RELAY_HOST is required
    set /a ERRORS+=1
)

if not defined AGENT_ID (
    echo ❌ AGENT_ID is required
    set /a ERRORS+=1
)

if not defined AGENT_RELAY_URL (
    echo ❌ AGENT_RELAY_URL is required
    set /a ERRORS+=1
)

REM Check if at least one service is configured
set SERVICES_COUNT=0
if defined AGENT_ALLOW_SSH set /a SERVICES_COUNT+=1
if defined AGENT_ALLOW_HTTP set /a SERVICES_COUNT+=1
if defined AGENT_ALLOW_HTTPS set /a SERVICES_COUNT+=1
if defined AGENT_ALLOW_WEB set /a SERVICES_COUNT+=1
if defined AGENT_ALLOW_DEV set /a SERVICES_COUNT+=1
if defined AGENT_ALLOW_POSTGRES set /a SERVICES_COUNT+=1
if defined AGENT_ALLOW_MYSQL set /a SERVICES_COUNT+=1
if defined AGENT_ALLOW_REDIS set /a SERVICES_COUNT+=1

if %SERVICES_COUNT%==0 (
    echo ⚠️  No services configured in AGENT_ALLOW_* variables
    echo    Services will be prompted during agent startup
) else (
    echo ✅ %SERVICES_COUNT% service(s) pre-configured
)

echo.
if %ERRORS%==0 (
    echo ✅ Configuration validation passed!
    echo.
    echo 🚀 Ready to start:
    echo 1. SSH Agent Terminal: start-ssh-agent-terminal.bat
    echo 2. For Linux server: ./start-relay.sh and ./start-agent.sh
) else (
    echo ❌ Configuration has %ERRORS% error(s)
    echo Please fix the errors before starting services
    pause
    exit /b 1
)

echo.
echo 💡 Example URLs after startup:
if defined RELAY_HOST (
    if defined RELAY_PORT (
        echo - Agent connection: wss://%RELAY_HOST%:%RELAY_PORT%/ws/agent
        echo - Client connection: wss://%RELAY_HOST%:%RELAY_PORT%/ws/client
        echo - Health check: https://%RELAY_HOST%:%RELAY_PORT%/health
    ) else (
        echo - Agent connection: wss://%RELAY_HOST%:8443/ws/agent
        echo - Client connection: wss://%RELAY_HOST%:8443/ws/client
        echo - Health check: https://%RELAY_HOST%:8443/health
    )
)

echo.
echo ===============================================
echo Configuration test completed
echo ===============================================
pause
