@echo off
REM MySQL/MariaDB Agent script
REM This script starts agent specifically configured for MySQL/MariaDB access

echo ========================================
echo MySQL/MariaDB Agent Setup
echo ========================================
echo Relay Server: sh.adisaputra.online
echo Agent: Local Laptop (MySQL/MariaDB enabled)
echo ========================================

REM Load configuration
if exist .env.production (
    echo Loading production configuration...
    for /f "tokens=1,2 delims==" %%a in (.env.production) do (
        if not "%%a"=="#" (
            set %%a=%%b
        )
    )
) else (
    echo Warning: .env.production not found, using defaults
    set TUNNEL_TOKEN=change-this-token
    set AGENT_ID=laptop-agent
    set AGENT_RELAY_URL=wss://sh.adisaputra.online:8443/ws/agent
)

echo.
echo Configuration:
echo - Token: %TUNNEL_TOKEN%
echo - Agent ID: %AGENT_ID%
echo - Relay URL: %AGENT_RELAY_URL%
echo.

REM Check if binaries exist
if not exist "bin\agent.exe" (
    echo Error: agent.exe not found. Building...
    call build.bat
    if errorlevel 1 (
        echo Build failed!
        pause
        exit /b 1
    )
)

echo MySQL/MariaDB Services Configuration:
echo [1] MySQL/MariaDB only (port 3306)
echo [2] MySQL + Web Server (ports 3306, 8080)
echo [3] MySQL + SSH (ports 3306, 22)
echo [4] All services including MySQL
set /p choice="Select option (1-4): "

if "%choice%"=="1" (
    set ALLOW_PORTS=-allow 127.0.0.1:3306
    echo Selected: MySQL/MariaDB only (port 3306)
) else if "%choice%"=="2" (
    set ALLOW_PORTS=-allow 127.0.0.1:3306 -allow 127.0.0.1:8080
    echo Selected: MySQL/MariaDB + Web Server (ports 3306, 8080)
) else if "%choice%"=="3" (
    set ALLOW_PORTS=-allow 127.0.0.1:3306 -allow 127.0.0.1:22
    echo Selected: MySQL/MariaDB + SSH (ports 3306, 22)
) else (
    set ALLOW_PORTS=-allow 127.0.0.1:3306 -allow 127.0.0.1:22 -allow 127.0.0.1:80 -allow 127.0.0.1:443 -allow 127.0.0.1:8080 -allow 127.0.0.1:5432
    echo Selected: All services including MySQL/MariaDB
)

echo.
echo ========================================
echo Starting MySQL/MariaDB agent:
echo.
echo Make sure MySQL/MariaDB is running and accessible on localhost:3306
echo.
echo Command: bin\agent.exe -id %AGENT_ID% -relay-url %AGENT_RELAY_URL% %ALLOW_PORTS% -token %TUNNEL_TOKEN% -insecure
echo.
echo Press Ctrl+C to stop the agent
echo ========================================

bin\agent.exe -id %AGENT_ID% -relay-url %AGENT_RELAY_URL% %ALLOW_PORTS% -token %TUNNEL_TOKEN% -insecure

echo.
echo MySQL/MariaDB agent stopped.
pause
