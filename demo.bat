@echo off
REM Demo script for Remote Tunnel

echo Starting Remote Tunnel Demo...
echo.

REM Set common token
set TUNNEL_TOKEN=demo-secret-token

REM Check if binaries exist
if not exist "bin\relay.exe" (
    echo Error: relay.exe not found. Please run build.bat first.
    pause
    exit /b 1
)

if not exist "bin\agent.exe" (
    echo Error: agent.exe not found. Please run build.bat first.
    pause
    exit /b 1
)

if not exist "bin\client.exe" (
    echo Error: client.exe not found. Please run build.bat first.
    pause
    exit /b 1
)

echo Token set to: %TUNNEL_TOKEN%
echo.

echo Instructions:
echo 1. First terminal will run the relay server
echo 2. Second terminal will run the agent
echo 3. Third terminal will run the client
echo 4. You can then test with: ssh -p 2222 localhost (if SSH server is running)
echo.
echo Press any key to start relay server...
pause > nul

REM Start relay server
echo Starting relay server on port 8443...
start "Relay Server" cmd /k "echo Relay Server && bin\relay.exe -addr :8443 -token %TUNNEL_TOKEN%"

REM Wait a bit for relay to start
timeout /t 3 > nul

echo.
echo Press any key to start agent...
pause > nul

REM Start agent
echo Starting agent...
start "Agent" cmd /k "echo Agent && bin\agent.exe -id demo-agent -relay-url wss://localhost:8443/ws/agent -allow 127.0.0.1: -token %TUNNEL_TOKEN%"

REM Wait a bit for agent to connect
timeout /t 3 > nul

echo.
echo Press any key to start client...
pause > nul

REM Start client
echo Starting client on port 2222...
start "Client" cmd /k "echo Client && bin\client.exe -L :2222 -relay-url wss://localhost:8443/ws/client -agent demo-agent -target 127.0.0.1:22 -token %TUNNEL_TOKEN%"

echo.
echo Demo started! All components should be running in separate windows.
echo.
echo To test the tunnel:
echo   ssh -p 2222 localhost
echo.
echo Or test with any TCP service running on port 22.
echo.
echo Press any key to exit...
pause > nul
