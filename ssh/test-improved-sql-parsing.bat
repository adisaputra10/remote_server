@echo off
echo ===============================================
echo Testing Improved SQL Parsing
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
start /b bin\universal-client.exe -n "MySQL Database Tunnel" -i mysql-client -t "103.41.206.153:3308" -a "agent-linux" -m tunnel -L ":3307"

echo.
echo Waiting for tunnel to establish...
timeout /t 5 /nobreak > nul

echo.
echo [Step 5] Running database test to generate various SQL operations...
go run .\test-db-golang.go

echo.
echo [Step 6] Checking logs...
echo.
echo === Agent Logs (last 20 lines) ===
type logs\AGENT-agent-linux.log | tail -20

echo.
echo === Relay Server Logs (last 20 lines) ===
type logs\server-relay.log | tail -20

echo.
echo [Step 7] Cleanup...
taskkill /f /im relay.exe > nul 2>&1
taskkill /f /im universal-client.exe > nul 2>&1
taskkill /f /im agent.exe > nul 2>&1

echo.
echo ===============================================
echo Test completed. 
echo Check dashboard Database Queries for improved parsing:
echo - TRUNCATE should show operation=TRUNCATE, table=test_users
echo - Other operations should be parsed correctly
echo ===============================================
pause