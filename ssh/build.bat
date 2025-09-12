@echo off
REM Build script for Tunnel System

echo 🚀 Building Tunnel System...

REM Create bin directory
if not exist "bin" mkdir bin
if not exist "logs" mkdir logs

REM Download dependencies
echo 📦 Downloading dependencies...
go mod download
go mod tidy

REM Build relay server
echo 🏗️ Building relay server...
go build -o bin\tunnel-relay.exe .\cmd\relay

REM Build tunnel agent
echo 🏗️ Building tunnel agent...
go build -o bin\tunnel-agent.exe .\cmd\agent

REM Build tunnel client
echo 🏗️ Building tunnel client...
go build -o bin\tunnel-client.exe .\cmd\client

echo ✅ Build complete!
echo.
echo 📂 Binaries created in bin\ directory:
echo   - tunnel-relay.exe
echo   - tunnel-agent.exe  
echo   - tunnel-client.exe
echo.
echo 🚀 Usage Examples:
echo.
echo 1️⃣ Start Relay Server (HTTPS with self-signed cert):
echo    bin\tunnel-relay.exe -addr :8443
echo.
echo 1️⃣ Start Relay Server (HTTP - INSECURE):
echo    bin\tunnel-relay.exe -addr :8080 -insecure
echo.
echo 2️⃣ Start Agent (secure):
echo    bin\tunnel-agent.exe -id my-agent -name "My Server" -relay-url wss://relay-server:8443/ws/agent
echo.
echo 2️⃣ Start Agent (insecure):
echo    bin\tunnel-agent.exe -id my-agent -name "My Server" -relay-url ws://relay-server:8080/ws/agent -insecure
echo.
echo 3️⃣ Use Client (interactive mode - secure):
echo    bin\tunnel-client.exe -relay-url wss://relay-server:8443/ws/client -i
echo.
echo 3️⃣ Use Client (interactive mode - insecure):
echo    bin\tunnel-client.exe -relay-url ws://relay-server:8080/ws/client -i -insecure
echo.
echo 4️⃣ Use Client (direct tunnel - secure):
echo    bin\tunnel-client.exe -L :2222 -agent my-agent -target 127.0.0.1:22 -relay-url wss://relay-server:8443/ws/client
echo.
echo 4️⃣ Use Client (direct tunnel - insecure):
echo    bin\tunnel-client.exe -L :2222 -agent my-agent -target 127.0.0.1:22 -relay-url ws://relay-server:8080/ws/client -insecure
echo.
echo 📋 Common Targets:
echo    SSH:        127.0.0.1:22
echo    MySQL:      127.0.0.1:3306
echo    PostgreSQL: 127.0.0.1:5432
echo.
echo 📊 Monitoring:
echo    Health:     http://relay-server:8443/health
echo    Agents:     http://relay-server:8443/api/agents
echo    Tunnels:    http://relay-server:8443/api/tunnels
echo.
pause
