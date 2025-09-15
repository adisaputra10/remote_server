@echo off
echo Building SSH tunnel client...

echo.
echo 1. Building SSH client...
go build -o bin/ssh-client.exe cmd/ssh-client/main.go

echo.
echo 2. Building relay server...
go build -o bin/relay.exe cmd/relay/main.go

echo.
echo 3. Building agent...
go build -o bin/agent.exe cmd/agent/main.go

echo.
echo 4. Building normal client...
go build -o bin/client.exe cmd/client/main.go

echo.
echo Build completed! Files created in bin/ directory:
dir bin\

echo.
echo Usage examples:
echo.
echo Start relay server:
echo   bin\relay.exe --port 8080
echo.
echo Start agent:
echo   bin\agent.exe --agent-id ssh-agent --relay ws://localhost:8080/ws/agent
echo.
echo Start SSH tunnel client:
echo   bin\ssh-client.exe --client-id ssh-client-1 --agent ssh-agent --ssh-host 192.168.1.100 --ssh-user root --local-port 2222
echo.
echo Connect via SSH:
echo   ssh root@localhost -p 2222
echo.
echo Check SSH logs in dashboard:
echo   http://localhost:8080 (or frontend)