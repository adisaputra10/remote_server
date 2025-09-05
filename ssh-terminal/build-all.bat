@echo off
echo Building GoTeleport with Database Proxy Support...
echo.

REM Build Agent
echo Building Agent (goteleport-agent.exe)...
go build -o goteleport-agent.exe goteleport-agent.go
if errorlevel 1 (
    echo Error: Failed to build agent
    goto :error
)
echo ✅ Agent built successfully

REM Build Server
echo Building Server (goteleport-server-db.exe)...
go build -o goteleport-server-db.exe goteleport-server-db.go
if errorlevel 1 (
    echo Error: Failed to build server
    goto :error
)
echo ✅ Server built successfully

REM Build Interactive Client
echo Building Interactive Client (interactive-client-clean.exe)...
go build -o interactive-client-clean.exe interactive-client-clean.go
if errorlevel 1 (
    echo Error: Failed to build client
    goto :error
)
echo ✅ Interactive Client built successfully

echo.
echo 🎉 All components built successfully!
echo.
echo Next steps:
echo 1. Configure your database proxy in agent-config-db.json
echo 2. Start the server: goteleport-server-db.exe server-config-db.json
echo 3. Start the agent: start-db-agent.bat
echo 4. Test database proxy: test-db-proxy.bat
echo.
echo For documentation, see DATABASE-PROXY.md
echo.
goto :end

:error
echo.
echo ❌ Build failed! Please check the errors above.
echo Make sure all dependencies are installed:
echo   go mod tidy
echo.

:end
pause
