@echo off
:: Quick Start SSH Agent Terminal
:: Script cepat untuk menjalankan terminal SSH agent interaktif

echo ===============================================
echo Quick Start SSH Agent Terminal
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

echo ✓ SSH Agent Terminal ready
echo.
echo Starting interactive SSH agent terminal...
echo.
echo ===============================================
echo SSH Agent Terminal Commands:
echo ===============================================
echo.
echo Connection Management:
echo  start-relay    - Start relay server
echo  start-agent    - Start SSH agent with forwarding
echo  restart-all    - Restart all services
echo.
echo SSH Operations:
echo  ssh-test       - Test SSH connectivity
echo  ssh-connect    - Interactive SSH session
echo  ssh-exec cmd   - Execute remote SSH command
echo.
echo System Monitoring:
echo  status         - Show system status
echo  check-ssh      - Check SSH service
echo.
echo Help:
echo  help           - Show all commands
echo  exit           - Quit terminal
echo.
echo ===============================================

D:\repo\remote\ssh-agent-terminal\ssh-agent-terminal.exe

echo.
echo SSH Agent Terminal session ended.
pause
