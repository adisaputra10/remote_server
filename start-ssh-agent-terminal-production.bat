@echo off
:: SSH Agent Terminal - Production Mode Quick Start
:: Script untuk langsung menjalankan terminal dalam mode production

echo ===============================================
echo SSH Agent Terminal - Production Mode
echo ===============================================
echo.
echo 🌐 Target Server: sh.adisaputra.online:8443
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

echo ✓ SSH Agent Terminal ready for production
echo.
echo ===============================================
echo Production Configuration Required:
echo ===============================================
echo.
echo You will be prompted for:
echo  1. Server domain (sh.adisaputra.online)
echo  2. Agent ID (server-agent)
echo  3. Security token
echo  4. SSH target host
echo  5. SSH port (22)
echo  6. SSH username
echo  7. SSH password
echo.
echo ===============================================
echo Production Mode Commands:
echo ===============================================
echo.
echo Connection:
echo  start-agent    - Connect to production relay
echo  ssh-test       - Test SSH connectivity to server
echo  ssh-connect    - SSH into production server
echo  status         - Check connection status
echo.
echo Configuration:
echo  config         - Show current settings
echo  reconnect      - Change configuration
echo.
echo ===============================================

echo Starting Production SSH Agent Terminal...
echo.
echo NOTE: Select "2" for Production Mode when prompted
echo.

D:\repo\remote\ssh-agent-terminal\ssh-agent-terminal.exe

echo.
echo Production SSH Agent Terminal session ended.
pause
