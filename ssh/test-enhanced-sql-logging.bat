@echo off
echo ===============================================
echo Testing Enhanced SQL Logging with DBeaver Support
echo ===============================================

echo.
echo [Step 1] Cleaning up previous processes...
taskkill /f /im relay.exe > nul 2>&1
taskkill /f /im universal-client.exe > nul 2>&1
taskkill /f /im agent.exe > nul 2>&1
timeout /t 2 /nobreak > nul

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
echo [Step 5] Testing with Golang client (should work)...
go run .\test-db-golang.go

echo.
echo [Step 6] Now connect with DBeaver to localhost:3307 and run some queries
echo Press any key after you've run queries in DBeaver...
pause

echo.
echo [Step 7] Checking logs for SQL parsing...
echo.
echo === Recent Agent Logs (last 30 lines) ===
type logs\AGENT-agent-linux.log | tail -30

echo.
echo === Recent Relay Server Logs (last 15 lines) ===
type logs\server-relay.log | tail -15

echo.
echo [Step 8] Cleanup...
taskkill /f /im relay.exe > nul 2>&1
taskkill /f /im universal-client.exe > nul 2>&1
taskkill /f /im agent.exe > nul 2>&1

echo.
echo ===============================================
echo Test completed. Check dashboard Database Queries for all logged commands.
echo ===============================================
pause