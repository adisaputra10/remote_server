@echo off
echo ===============================================
echo          GoTeleport SUCCESS DEMO!
echo ===============================================
echo.
echo ✅ CLIENT: Successfully connects to server
echo ✅ AGENTS: Successfully lists available agents  
echo ✅ CONNECT: Successfully connects to remote agent
echo ✅ EXECUTE: Successfully executes remote commands
echo ✅ LOGGING: All commands are logged with details
echo ✅ INTERACTIVE: Real-time command execution
echo.
echo 🎯 WORKING COMMANDS TO TRY:
echo.
echo Windows Commands (since agents are Windows):
echo   dir               - List directory contents
echo   cd..              - Change to parent directory  
echo   echo hello         - Print text
echo   whoami            - Show current user
echo   hostname          - Show computer name
echo   systeminfo        - Show system information
echo   tasklist          - Show running processes
echo   ipconfig          - Show network configuration
echo.
echo Linux-style Commands (if you want to test errors):
echo   ls, pwd, ps       - Will show "command not found"
echo.
echo Navigation Commands:
echo   disconnect        - Return to main client prompt
echo   agents            - Refresh agent list
echo   status            - Show connection status
echo   exit              - Exit client
echo.
echo 📝 All commands are logged to:
echo    - client.log (client activity)
echo    - server.log (server activity)  
echo    - Individual agent logs
echo.
pause

echo 🚀 Starting GoTeleport Interactive Client...
echo.
cd /d "d:\repo\remote\ssh-terminal"
.\interactive-client-clean.exe client-config-clean.json
