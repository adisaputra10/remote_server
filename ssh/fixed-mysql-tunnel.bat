@echo off
echo ╔════════════════════════════════════════════════════════════════════╗
echo ║          🎯 Fixed Command - Database Tunnel with Client ID       ║
echo ╚════════════════════════════════════════════════════════════════════╝
echo.
echo Original command (without client identification):
echo bin\universal-client.exe -L ":3307" -a "agent-linux" -t "103.41.206.153:3308"
echo.
echo ❌ Problem: CLIENT ID shows as "-" in dashboard
echo.
echo ✅ FIXED COMMAND (with client identification):
echo bin\universal-client.exe -L ":3307" -a "agent-linux" -t "103.41.206.153:3308" -c "mysql-tunnel-client" -n "MySQL Database Tunnel"
echo.
echo Parameters added:
echo   -c "mysql-tunnel-client"    = Client ID (will show in CLIENT ID column)
echo   -n "MySQL Database Tunnel"  = Client Name (descriptive name)
echo.
echo Running the fixed command...
echo ────────────────────────────────────────────────────────────────────
bin\universal-client.exe -L ":3307" -a "agent-linux" -t "103.41.206.153:3308" -c "mysql-tunnel-client" -n "MySQL Database Tunnel"