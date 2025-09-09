@echo off
echo 🔍 Testing connection to server (agent behind NAT)...

echo.
echo ℹ️  Network Architecture:
echo    Windows Client -> Server (168.231.119.242) -> Agent (behind NAT)
echo    
echo    - Agent is behind NAT and not directly accessible
echo    - All connections go through server as proxy
echo    - Server forwards requests to agent

echo.
echo Reading server configuration...
for /f "tokens=2 delims=:" %%i in ('findstr "server_url" client-config-clean.json') do (
    set server_line=%%i
)

echo Server URL from config: 
type client-config-clean.json | findstr server_url

echo.
echo Testing connection to SERVER (not agent directly):

echo.
echo 🔌 Testing server WebSocket port (8081)...
telnet 168.231.119.242 8081 2>nul || echo ❌ Server WebSocket port 8081 not accessible

echo.
echo � Note: We do NOT test agent ports directly because:
echo    - Agent is behind NAT (not directly accessible)
echo    - Agent ports (3307, 5435) are only accessible through server tunneling
echo    - Unified client will create tunnels through server to reach agent

echo.
echo 💡 If server port 8081 is not accessible:
echo    1. Check if server is running on Linux (168.231.119.242:8081)
echo    2. Check firewall rules on Linux server
echo    3. Verify server binds to 0.0.0.0:8081 (not 127.0.0.1)
echo    4. Check network connectivity to server

echo.
echo 🚀 Running unified client (will tunnel through server)...
.\unified-client.exe client-config-clean.json
