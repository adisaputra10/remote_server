@echo off
echo ╔════════════════════════════════════════════════════════════════════╗
echo ║            🔗 Universal Client - Database Tunnel Examples        ║
echo ╚════════════════════════════════════════════════════════════════════╝
echo.
echo Database tunnel examples with clear client identification:
echo.

echo 1. MySQL Database Tunnel with Client ID
echo ────────────────────────────────────────────────────────────────────
echo Command: bin\universal-client.exe -L ":3307" -a "agent-linux" -t "103.41.206.153:3308" -c "mysql-client" -n "MySQL Database Tunnel"
echo.
pause
bin\universal-client.exe -L ":3307" -a "agent-linux" -t "103.41.206.153:3308" -c "mysql-client" -n "MySQL Database Tunnel"
pause

echo.
echo 2. PostgreSQL Database Tunnel with Client ID
echo ────────────────────────────────────────────────────────────────────
echo Command: bin\universal-client.exe -L ":5433" -a "agent-linux" -t "103.41.206.153:5432" -c "postgres-client" -n "PostgreSQL Database Tunnel"
echo.
pause
bin\universal-client.exe -L ":5433" -a "agent-linux" -t "103.41.206.153:5432" -c "postgres-client" -n "PostgreSQL Database Tunnel"
pause

echo.
echo 3. Web Application Tunnel with Client ID
echo ────────────────────────────────────────────────────────────────────
echo Command: bin\universal-client.exe -L ":8081" -a "agent-linux" -t "103.41.206.153:80" -c "web-client" -n "Web Application Tunnel"
echo.
pause
bin\universal-client.exe -L ":8081" -a "agent-linux" -t "103.41.206.153:80" -c "web-client" -n "Web Application Tunnel"
pause

echo.
echo Check the dashboard now - CLIENT ID column should show the specified names!
pause