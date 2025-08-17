@echo off
REM Production setup script for Windows laptop (Agent)
REM Relay Server: 103.195.169.32

echo ========================================
echo Remote Tunnel - Laptop Agent Setup
echo ========================================
echo Relay Server: sh.adisaputra.online
echo Agent: Local Laptop
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

echo Services to expose:
echo [1] SSH Server (port 22)
echo [2] Web Server (port 8080)  
echo [3] Database (port 5432)
echo [4] Custom ports
echo [5] All common services
set /p choice="Select option (1-5): "

if "%choice%"=="1" (
    set ALLOW_PORTS=-allow 127.0.0.1:22
    echo Selected: SSH Server only
) else if "%choice%"=="2" (
    set ALLOW_PORTS=-allow 127.0.0.1:8080
    echo Selected: Web Server only
) else if "%choice%"=="3" (
    set ALLOW_PORTS=-allow 127.0.0.1:5432
    echo Selected: Database only
) else if "%choice%"=="4" (
    set /p custom_ports="Enter ports (e.g., 127.0.0.1:3000 127.0.0.1:8000): "
    set ALLOW_PORTS=-allow %custom_ports%
    echo Selected: Custom ports
) else (
    set ALLOW_PORTS=-allow 127.0.0.1:22 -allow 127.0.0.1:80 -allow 127.0.0.1:443 -allow 127.0.0.1:3000 -allow 127.0.0.1:8080 -allow 127.0.0.1:5432
    echo Selected: All common services
)

echo.
echo Starting agent with configuration:
echo bin\agent.exe -id %AGENT_ID% -relay-url %AGENT_RELAY_URL% %ALLOW_PORTS% -token %TUNNEL_TOKEN%
echo.
echo Press Ctrl+C to stop the agent
echo ========================================

bin\agent.exe -id %AGENT_ID% -relay-url %AGENT_RELAY_URL% %ALLOW_PORTS% -token %TUNNEL_TOKEN%

echo.
echo Agent stopped.
pause
