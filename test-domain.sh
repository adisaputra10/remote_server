#!/bin/bash
# Test connection specifically for sh.adisaputra.online

echo "========================================"
echo "Remote Tunnel - Domain Connection Test"
echo "========================================"

DOMAIN="sh.adisaputra.online"
RELAY_PORT="8443"

echo "Testing connection to relay server..."
echo "Domain: $DOMAIN"
echo "Port: $RELAY_PORT"
echo

echo "[1/5] DNS Resolution Test..."
if nslookup "$DOMAIN" >/dev/null 2>&1; then
    echo "✅ PASS: Domain $DOMAIN resolves correctly"
else
    echo "❌ FAIL: Cannot resolve domain $DOMAIN"
    echo "Check DNS configuration"
fi

echo
echo "[2/5] Basic Connectivity Test..."
if ping -c 2 "$DOMAIN" >/dev/null 2>&1; then
    echo "✅ PASS: Relay server is reachable"
else
    echo "❌ FAIL: Cannot ping relay server"
    echo "Check internet connection and server status"
fi

echo
echo "[3/5] Port Connectivity Test..."
if timeout 10 bash -c "</dev/tcp/$DOMAIN/$RELAY_PORT" 2>/dev/null; then
    echo "✅ PASS: Port $RELAY_PORT is open"
else
    echo "❌ FAIL: Port $RELAY_PORT is closed or filtered"
fi

echo
echo "[4/5] TLS Certificate Test..."
if openssl s_client -connect "$DOMAIN:$RELAY_PORT" -servername "$DOMAIN" -verify_return_error </dev/null 2>/dev/null; then
    echo "✅ PASS: TLS connection and certificate valid"
elif openssl s_client -connect "$DOMAIN:$RELAY_PORT" -servername "$DOMAIN" </dev/null 2>/dev/null | grep -q "CONNECTED"; then
    echo "✅ PASS: TLS connection successful (self-signed certificate)"
else
    echo "❌ FAIL: TLS connection failed"
fi

echo
echo "[5/5] Relay Server Health Test..."
HTTP_CODE=$(curl -k -s -o /dev/null -w "%{http_code}" "https://$DOMAIN:$RELAY_PORT/health" --connect-timeout 10 2>/dev/null)
if [ "$HTTP_CODE" = "200" ]; then
    echo "✅ PASS: Relay server health endpoint responding"
else
    echo "❌ FAIL: Relay server health check failed (HTTP $HTTP_CODE)"
fi

echo
echo "[Bonus] Agent Binary Check..."
if [ -f "bin/agent" ]; then
    echo "✅ PASS: Agent binary found"
else
    echo "❌ FAIL: Agent binary not found - run build.sh first"
fi

echo
echo "========================================"
echo "Connection Test Results Summary"
echo "========================================"
echo
echo "If all tests pass, you can proceed with:"
echo
echo "1. Generate secure token:"
echo "   ./generate-token.sh"
echo
echo "2. Start agent on this laptop:"
echo "   ./start-agent.sh"
echo
echo "3. From remote machine, create client connection:"
echo "   ./bin/client -L :2222 -relay-url wss://$DOMAIN:$RELAY_PORT/ws/client -agent laptop-agent -target 127.0.0.1:22 -token YOUR_TOKEN"
echo
echo "4. Test SSH connection:"
echo "   ssh -p 2222 user@localhost"
echo
echo "Troubleshooting:"
echo "- If DNS fails: Check domain configuration"
echo "- If ping fails: Check internet/firewall"
echo "- If port fails: Check relay server status"
echo "- If TLS fails: Check certificate configuration"
echo "- If health fails: Check relay server logs"
echo

read -p "Press Enter to continue..."
