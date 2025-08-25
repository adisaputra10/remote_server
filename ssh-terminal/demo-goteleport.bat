@echo off
echo ===============================================
echo          GoTeleport Quick Test Demo
echo ===============================================
echo.
echo This script demonstrates the full GoTeleport workflow:
echo 1. Client connects to server
echo 2. Lists available agents  
echo 3. Connects to an agent
echo 4. Executes commands remotely
echo 5. Shows command logging
echo.

echo Instructions for testing:
echo.
echo In the client terminal that will open:
echo   1. Type "agents" to see available agents
echo   2. Type "connect 1" to connect to first agent
echo   3. Type any command like "dir", "whoami", "hostname"
echo   4. Type "disconnect" to return to client prompt
echo   5. Type "exit" to quit
echo.
echo Note: All commands are logged to files!
echo.
pause

cd /d "d:\repo\remote\ssh-terminal"

echo Starting interactive client...
echo.
.\interactive-client-fixed.exe client-config-clean.json
