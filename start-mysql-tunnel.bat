@echo off
REM MySQL/MariaDB tunnel script
REM This script creates a tunnel specifically for MySQL/MariaDB access

echo ========================================
echo MySQL/MariaDB Remote Tunnel
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
)

echo.
echo Available agents to connect to:
echo - %AGENT_ID% (your laptop)
echo.
set /p target_agent="Enter agent ID [%AGENT_ID%]: "
if "%target_agent%"=="" set target_agent=%AGENT_ID%

echo.
echo MySQL/MariaDB Configuration:
echo [1] Default MySQL/MariaDB (localhost:3306)
echo [2] Custom MySQL/MariaDB server
set /p mysql_choice="Select option (1-2): "

if "%mysql_choice%"=="1" (
    set TARGET_ADDR=127.0.0.1:3306
    set LOCAL_PORT=3306
    echo Selected: Default MySQL/MariaDB on localhost:3306
) else (
    set /p TARGET_ADDR="Enter MySQL server address (e.g., 192.168.1.100:3306): "
    set /p LOCAL_PORT="Enter local port [3306]: "
    if "%LOCAL_PORT%"=="" set LOCAL_PORT=3306
    echo Selected: Custom MySQL/MariaDB server %TARGET_ADDR%
)

echo.
echo ========================================
echo Starting MySQL/MariaDB tunnel:
echo Local port %LOCAL_PORT% -> Agent %target_agent% -> MySQL %TARGET_ADDR%
echo.
echo After connection established, you can connect using:
echo   mysql -h localhost -P %LOCAL_PORT% -u your_username -p
echo   or use any MySQL client with host: localhost, port: %LOCAL_PORT%
echo.
echo MySQL Workbench connection:
echo   Host: localhost
echo   Port: %LOCAL_PORT%
echo   Username: your_mysql_username
echo.
echo Press Ctrl+C to stop the tunnel
echo ========================================
echo.

bin\client.exe -L :%LOCAL_PORT% -relay-url wss://sh.adisaputra.online:8443/ws/client -agent %target_agent% -target %TARGET_ADDR% -token %TUNNEL_TOKEN% -insecure

echo.
echo MySQL/MariaDB tunnel stopped.
pause
