@echo off
REM Build script for Windows

echo Building Remote Tunnel binaries...

REM Create bin directory
if not exist "bin" mkdir bin

REM Download dependencies
echo Downloading dependencies...
go mod download
go mod tidy

REM Build binaries
echo Building relay...
go build -o bin\relay.exe .\cmd\relay

echo Building agent...
go build -o bin\agent.exe .\cmd\agent

echo Building client...
go build -o bin\client.exe .\cmd\client

echo Build complete!
echo.
echo Binaries created in bin\ directory:
echo - relay.exe
echo - agent.exe  
echo - client.exe
echo.
echo To run a quick test:
echo 1. Set token: set TUNNEL_TOKEN=test-token
echo 2. Run relay: bin\relay.exe -addr :8443
echo 3. Run agent: bin\agent.exe -id test-agent -relay-url wss://localhost:8443/ws/agent -allow 127.0.0.1:22
echo 4. Run client: bin\client.exe -L :2222 -relay-url wss://localhost:8443/ws/client -agent test-agent -target 127.0.0.1:22
