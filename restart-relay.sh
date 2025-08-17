#!/bin/bash
# Restart relay server with port 8443

echo "========================================"
echo "Restarting Relay Server (Port 8443)"
echo "========================================"

echo "[1/3] Stopping existing relay..."
./stop-relay.sh

echo
echo "[2/3] Verifying configuration..."

# Check .env.production
if [ -f ".env.production" ]; then
    echo "✅ .env.production found"
    
    # Check for correct port configuration
    if grep -q "RELAY_ADDR=:8443" .env.production; then
        echo "✅ RELAY_ADDR is set to :8443"
    else
        echo "⚠️  Updating RELAY_ADDR to :8443"
        sed -i 's/RELAY_ADDR=.*/RELAY_ADDR=:8443/' .env.production
    fi
    
    if grep -q "RELAY_PORT=8443" .env.production; then
        echo "✅ RELAY_PORT is set to 8443"
    else
        echo "⚠️  Updating RELAY_PORT to 8443"
        sed -i 's/RELAY_PORT=.*/RELAY_PORT=8443/' .env.production
    fi
else
    echo "❌ .env.production not found"
    exit 1
fi

# Check if binary exists
if [ ! -f "bin/relay" ]; then
    echo "❌ Relay binary not found. Building..."
    make build-linux
    if [ $? -ne 0 ]; then
        echo "❌ Build failed"
        exit 1
    fi
    echo "✅ Build successful"
else
    echo "✅ Relay binary found"
fi

echo
echo "[3/3] Starting relay on port 8443..."

# Start relay server
./start-relay.sh &
RELAY_PID=$!

# Wait a moment and check if it started
sleep 3

if kill -0 $RELAY_PID 2>/dev/null; then
    echo "✅ Relay server started successfully (PID: $RELAY_PID)"
    echo
    echo "Verifying port 8443..."
    sleep 2
    
    if netstat -tlnp 2>/dev/null | grep ":8443.*LISTEN"; then
        echo "✅ Relay server is listening on port 8443"
    else
        echo "❌ Relay server is not listening on port 8443"
        echo "Check logs for errors"
    fi
else
    echo "❌ Relay server failed to start"
    echo "Check logs for errors"
fi

echo
echo "========================================"
echo "Restart Complete"
echo "========================================"
echo
echo "Commands:"
echo "- Check status: netstat -tlnp | grep 8443"
echo "- View logs: tail -f relay.log"
echo "- Stop server: ./stop-relay.sh"
echo "- Test health: curl -k https://sh.adisaputra.online:8443/health"
