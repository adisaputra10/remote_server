#!/bin/bash
# Check relay server status and ports

echo "========================================"
echo "Remote Tunnel Relay Status Check"
echo "========================================"

echo "[1] Process Status:"
RELAY_PIDS=$(pgrep -f "bin/relay" 2>/dev/null)
if [ -n "$RELAY_PIDS" ]; then
    echo "✅ Relay processes running:"
    for PID in $RELAY_PIDS; do
        echo "   PID $PID: $(ps -p $PID -o cmd --no-headers 2>/dev/null)"
    done
else
    echo "❌ No relay processes found"
fi

echo
echo "[2] Port Status:"

# Check port 443
PORT_443=$(netstat -tlnp 2>/dev/null | grep ":443.*LISTEN")
if [ -n "$PORT_443" ]; then
    echo "⚠️  Port 443 is in use:"
    echo "   $PORT_443"
else
    echo "✅ Port 443 is free"
fi

# Check port 8443
PORT_8443=$(netstat -tlnp 2>/dev/null | grep ":8443.*LISTEN")
if [ -n "$PORT_8443" ]; then
    echo "✅ Port 8443 is in use:"
    echo "   $PORT_8443"
else
    echo "❌ Port 8443 is free"
fi

echo
echo "[3] Configuration Check:"
if [ -f ".env.production" ]; then
    echo "✅ .env.production exists:"
    echo "   RELAY_ADDR: $(grep RELAY_ADDR .env.production | cut -d'=' -f2)"
    echo "   RELAY_PORT: $(grep RELAY_PORT .env.production | cut -d'=' -f2)"
    echo "   TUNNEL_TOKEN: $(grep TUNNEL_TOKEN .env.production | cut -d'=' -f2 | cut -c1-10)..."
else
    echo "❌ .env.production not found"
fi

echo
echo "[4] Binary Check:"
if [ -f "bin/relay" ]; then
    echo "✅ Relay binary exists ($(stat -c%s bin/relay) bytes)"
else
    echo "❌ Relay binary not found"
fi

echo
echo "[5] Certificate Check:"
if [ -f "certs/server.crt" ] && [ -f "certs/server.key" ]; then
    echo "✅ Self-signed certificates exist:"
    echo "   Certificate: certs/server.crt"
    echo "   Private Key: certs/server.key"
    echo "   Valid until: $(openssl x509 -in certs/server.crt -noout -enddate 2>/dev/null | cut -d'=' -f2)"
else
    echo "❌ Self-signed certificates not found"
fi

echo
echo "[6] Connectivity Test:"
if command -v curl >/dev/null 2>&1; then
    echo -n "Testing health endpoint... "
    HTTP_CODE=$(curl -k -s -o /dev/null -w "%{http_code}" "https://sh.adisaputra.online:8443/health" --connect-timeout 5 2>/dev/null)
    if [ "$HTTP_CODE" = "200" ]; then
        echo "✅ Health endpoint responds (HTTP $HTTP_CODE)"
    else
        echo "❌ Health endpoint failed (HTTP $HTTP_CODE)"
    fi
else
    echo "⚠️  curl not available for testing"
fi

echo
echo "========================================"
echo "Status Summary"
echo "========================================"

if [ -n "$RELAY_PIDS" ] && [ -n "$PORT_8443" ]; then
    echo "🟢 Relay server is running correctly on port 8443"
elif [ -n "$RELAY_PIDS" ] && [ -n "$PORT_443" ]; then
    echo "🟡 Relay server is running but on port 443 (should be 8443)"
    echo "   Run: ./restart-relay.sh"
elif [ -n "$RELAY_PIDS" ]; then
    echo "🟡 Relay process running but port unclear"
elif [ -n "$PORT_8443" ]; then
    echo "🟡 Port 8443 in use but no relay process found"
else
    echo "🔴 Relay server is not running"
    echo "   Run: ./start-relay.sh"
fi

echo
echo "Commands:"
echo "- Start: ./start-relay.sh"
echo "- Stop:  ./stop-relay.sh" 
echo "- Restart: ./restart-relay.sh"
echo "- Monitor: ./monitor-connection.sh"
