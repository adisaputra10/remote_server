@echo off
:: SSH Agent Terminal v2.0 - Local & Production Support
:: Script untuk menjalankan terminal SSH agent interaktif

echo ===============================================
echo SSH Agent Terminal v2.0
echo ===============================================
echo.

echo Checking SSH agent terminal...
if not exist "ssh-agent-terminal\ssh-agent-terminal.exe" (
    echo Building SSH agent terminal...
    cd ssh-agent-terminal
    go build -o ssh-agent-terminal.exe .
    cd ..
    
    if not exist "ssh-agent-terminal\ssh-agent-terminal.exe" (
        echo ✗ Failed to build SSH agent terminal
        pause
        exit /b 1
    )
)

echo ✓ SSH Agent Terminal v2.0 ready
echo.
echo ===============================================
echo Available Modes:
echo ===============================================
echo.
echo 🏠 Local Mode (Development):
echo   - Local relay server (localhost:8080)
echo   - Local SSH server (john@127.0.0.1:22)
echo   - Self-signed certificates
echo   - Default credentials
echo.
echo 🌐 Production Mode:
echo   - Remote relay server (sh.adisaputra.online:8443)
echo   - Remote SSH server (configurable)
echo   - Custom credentials via prompts
echo   - Production certificates
echo.
echo ===============================================
echo Terminal will prompt for:
echo ===============================================
echo.
echo Mode Selection:
echo  1 = Local Development Mode
echo  2 = Production Server Mode
echo.
echo For Production Mode, you'll configure:
echo  - Server domain (default: sh.adisaputra.online)
echo  - Agent ID (default: server-agent)
echo  - Security token
echo  - SSH host and port
echo  - SSH username and password
echo.
echo ===============================================
echo SSH Agent Terminal Commands:
echo ===============================================
echo.
echo Connection Management:
echo  start-relay    - Start relay server (local only)
echo  start-agent    - Start/connect SSH agent
echo  restart-all    - Restart all services
echo  reconnect      - Reconfigure and reconnect
echo.
echo SSH Operations:
echo  ssh-test       - Test SSH connectivity
echo  ssh-connect    - Interactive SSH session
echo  ssh-exec cmd   - Execute remote SSH command
echo.
echo System Monitoring:
echo  status         - Show system status
echo  config         - Show current configuration
echo  check-ssh      - Check SSH service
echo.
echo Help:
echo  help           - Show all commands
echo  exit           - Quit terminal
echo.
echo ===============================================

echo Starting SSH Agent Terminal v2.0...
echo.

D:\repo\remote\ssh-agent-terminal\ssh-agent-terminal.exe

echo.
echo SSH Agent Terminal session ended.
pause
