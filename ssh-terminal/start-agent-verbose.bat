@echo off
echo 🚀 Starting GoTeleport Agent with Verbose Logging...

echo.
echo 📦 Building agent...
go build -o goteleport-agent-db.exe goteleport-agent-db.go

if %ERRORLEVEL% NEQ 0 (
    echo ❌ Failed to build agent
    pause
    exit /b 1
)

echo ✅ Agent built successfully

echo.
echo 📋 Agent config:
type agent-config-db.json

echo.
echo 🔧 Starting agent with verbose output...
echo 📝 Logs will be written to both agent-db.log and stdout
echo.
echo Press Ctrl+C to stop agent
echo.

REM Start agent
goteleport-agent-db.exe agent-config-db.json
