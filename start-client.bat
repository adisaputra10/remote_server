@echo off
REM Client script for testing connection through relay
REM Use this to test from another machine

echo ========================================
echo Remote Tunnel - Test Client
echo ========================================
echo Relay Server: sh.adisaputra.online
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
    set CLIENT_RELAY_URL=wss://sh.adisaputra.online:8443/ws/client
)

echo.
echo Available agents to connect to:
echo - %AGENT_ID% (your laptop)
echo.
set /p target_agent="Enter agent ID [%AGENT_ID%]: "
if "%target_agent%"=="" set target_agent=%AGENT_ID%

echo.
echo Common target services:
echo [1] SSH (port 22)
echo [2] Web Server (port 8080)
echo [3] PostgreSQL (port 5432)
echo [4] MySQL/MariaDB (port 3306)
echo [5] Custom
set /p service_choice="Select service (1-5): "

if "%service_choice%"=="1" (
    set TARGET_ADDR=127.0.0.1:22
    set LOCAL_PORT=2222
    echo Selected: SSH - Access via ssh -p 2222 user@localhost
) else if "%service_choice%"=="2" (
    set TARGET_ADDR=127.0.0.1:8080
    set LOCAL_PORT=8080
    echo Selected: Web Server - Access via http://localhost:8080
) else if "%service_choice%"=="3" (
    set TARGET_ADDR=127.0.0.1:5432
    set LOCAL_PORT=5432
    echo Selected: PostgreSQL - Access via localhost:5432
) else if "%service_choice%"=="4" (
    set TARGET_ADDR=127.0.0.1:3306
    set LOCAL_PORT=3306
    echo Selected: MySQL/MariaDB - Access via localhost:3306
    echo Example: mysql -h localhost -P 3306 -u username -p
) else (
    set /p TARGET_ADDR="Enter target address (e.g., 127.0.0.1:3000): "
    set /p LOCAL_PORT="Enter local port (e.g., 3000): "
    echo Selected: Custom service
)

set COMPRESSION_FLAG=


echo.
echo Starting client tunnel:
echo Local port %LOCAL_PORT% -> Agent %target_agent% -> Target %TARGET_ADDR%
echo.
echo Command: bin\client.exe -L :%LOCAL_PORT% -relay-url wss://sh.adisaputra.online:8443/ws/client  -agent %target_agent% -target %TARGET_ADDR% -token %TUNNEL_TOKEN% -insecure %COMPRESSION_FLAG%
echo.
echo Press Ctrl+C to stop
echo ========================================

bin\client.exe -L :%LOCAL_PORT% -relay-url wss://sh.adisaputra.online:8443/ws/client  -agent %target_agent% -target %TARGET_ADDR% -token %TUNNEL_TOKEN% -insecure %COMPRESSION_FLAG%

echo.
echo Client stopped.
pause
