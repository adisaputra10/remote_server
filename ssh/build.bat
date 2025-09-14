@echo off
REM Build script for SSH Tunnel components

echo Building SSH Tunnel components...

REM Create bin directory if it doesn't exist
if not exist "bin" mkdir bin

REM Build Relay Server
echo Building Relay Server...
go build -o bin/tunnel-relay.exe ./cmd/relay
if %ERRORLEVEL% neq 0 (
    echo Failed to build relay server
    exit /b 1
)
echo Relay Server built successfully: bin/tunnel-relay.exe

REM Build Agent
echo Building Agent...
go build -o bin/tunnel-agent.exe ./cmd/agent
if %ERRORLEVEL% neq 0 (
    echo Failed to build agent
    exit /b 1
)
echo Agent built successfully: bin/tunnel-agent.exe

REM Build Client
echo Building Client...
go build -o bin/tunnel-client.exe ./cmd/client
if %ERRORLEVEL% neq 0 (
    echo Failed to build client
    exit /b 1
)
echo Client built successfully: bin/tunnel-client.exe

echo.
echo All components built successfully!
echo.
echo Available executables:
echo   bin/tunnel-relay.exe  - Relay server
echo   bin/tunnel-agent.exe  - SSH agent
echo   bin/tunnel-client.exe - Tunnel client
echo.
echo Run with -h flag to see usage options for each component.