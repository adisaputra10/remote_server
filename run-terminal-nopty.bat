@echo off
:: SSH Agent Terminal (No PTY) - Interactive Go Terminal
:: Terminal interaktif Go untuk SSH agent access tanpa PTY

echo ===============================================
echo SSH Agent Terminal (No PTY)
echo ===============================================
echo.

echo Step 1: Building SSH Agent Terminal
echo ===================================
echo Building Go terminal with SSH agent integration...

if not exist "ssh-agent-terminal" (
    echo ✗ SSH Agent Terminal directory not found
    echo Creating directory...
    mkdir ssh-agent-terminal
)

cd ssh-agent-terminal

echo Building SSH agent terminal executable...
go build -o ssh-agent-terminal.exe .

if %ERRORLEVEL%==0 (
    echo ✓ SUCCESS: SSH Agent Terminal built successfully
) else (
    echo ✗ ERROR: Failed to build SSH agent terminal
    echo Make sure Go is installed and source files exist
    cd ..
    pause
    exit /b 1
)
echo.

echo Step 2: SSH Agent Terminal Features
echo ===================================
echo.
echo This interactive terminal provides:
echo ✓ SSH agent management (start/stop)
echo ✓ Relay server control
echo ✓ Direct SSH connections (no PTY)
echo ✓ SSH connectivity testing
echo ✓ System status monitoring
echo ✓ Built-in SSH commands
echo.

echo Step 3: Available Commands
echo ==========================
echo.
echo Connection Management:
echo  start-relay   - Start relay server (port 8080)
echo  start-agent   - Start SSH agent with forwarding
echo  stop-relay    - Stop relay server
echo  stop-agent    - Stop SSH agent
echo  restart-all   - Restart all services
echo.
echo SSH Operations:
echo  ssh-connect   - Interactive SSH session
echo  ssh-test      - Test SSH connectivity
echo  ssh-exec cmd  - Execute SSH command
echo.
echo System Checks:
echo  status        - Show system status
echo  check-ssh     - Check SSH service
echo  check-port 22 - Check port availability
echo.

echo Step 4: SSH Configuration
echo =========================
echo.
echo Target SSH server:
echo  Host: 127.0.0.1 (localhost)
echo  Port: 22
echo  User: john
echo  Password: john123
echo.
echo Agent configuration:
echo  Relay: wss://localhost:8080/ws/agent
echo  Agent ID: demo-agent
echo  Token: demo-token
echo  Forwarding: 127.0.0.1:22
echo.

echo Step 5: Starting SSH Agent Terminal
echo ===================================
echo.
choice /c YN /m "Start SSH Agent Terminal"
if errorlevel 2 goto :skip_start
if errorlevel 1 goto :start_terminal

:start_terminal
echo.
echo Starting SSH Agent Terminal...
echo.
echo ===============================================
echo Welcome to SSH Agent Terminal!
echo ===============================================
echo.
echo Quick Start Guide:
echo 1. Type 'status' to check system
echo 2. Type 'start-relay' to start relay server
echo 3. Type 'start-agent' to start SSH agent
echo 4. Type 'ssh-test' to test SSH connection
echo 5. Type 'ssh-connect' for interactive SSH
echo 6. Type 'help' for all commands
echo.

REM Start the SSH agent terminal
.\ssh-agent-terminal.exe

goto :end

:skip_start
echo Terminal build completed but not started.
echo.
echo To start SSH agent terminal manually:
echo   cd ssh-agent-terminal
echo   .\ssh-agent-terminal.exe
echo.

:end
cd ..
echo.
echo ===============================================
echo SSH Agent Terminal Session Ended
echo ===============================================
echo.
echo SSH Agent Terminal with SSH remote access
echo is ready for use.
echo.
echo Key features:
echo ✓ No PTY client needed
echo ✓ Direct SSH connections
echo ✓ Interactive command interface
echo ✓ Agent and relay management
echo ✓ Built-in SSH operations
echo ✓ System monitoring
echo.
echo Use this terminal to manage SSH agent
echo and access remote SSH server easily!
echo.
pause
