@echo off
echo ===============================================
echo Testing DBeaver Database Logging
echo ===============================================

echo.
echo [Step 1] Cleaning up previous processes...
taskkill /f /im relay.exe > nul 2>&1
taskkill /f /im universal-client.exe > nul 2>&1
taskkill /f /im agent.exe > nul 2>&1
timeout /t 3 /nobreak > nul

echo.
echo [Step 2] Starting relay server...
start /min bin\relay.exe
echo Waiting for relay server to initialize...
timeout /t 5 /nobreak > nul

echo.
echo [Step 3] Starting agent...
start /min bin\agent.exe -id agent-linux
timeout /t 3 /nobreak > nul

echo.
echo [Step 4] Starting tunnel client...
echo Target: 103.41.206.153:3308
echo Client ID: mysql-client
echo Agent ID: agent-linux

start /b bin\universal-client.exe -n "MySQL Database Tunnel" -i mysql-client -t "103.41.206.153:3308" -a "agent-linux" -m tunnel -L ":3307"

echo.
echo Waiting for tunnel to establish...
timeout /t 8 /nobreak > nul

echo.
echo [Step 5] Running simple database test...
go run .\test-db-golang.go

echo.
echo [Step 6] Checking logs...
echo.
echo === Recent Agent Logs (last 20 lines) ===
type logs\AGENT-agent-linux.log | tail -20

echo.
echo === Recent Relay Server Logs (last 20 lines) ===
type logs\server-relay.log | tail -20

echo.
echo [Step 7] Cleanup...
taskkill /f /im relay.exe > nul 2>&1
taskkill /f /im universal-client.exe > nul 2>&1
taskkill /f /im agent.exe > nul 2>&1

echo.
echo ===============================================
echo Test completed. Now try DBeaver connection to localhost:3307
echo and check the dashboard Database Queries tab.
echo ===============================================
pause