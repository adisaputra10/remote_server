#!/bin/bash
# Test script untuk memverifikasi setup (Linux version)

set -e

echo "========================================"
echo "Remote Tunnel - Connection Test"
echo "========================================"

RELAY_HOST="sh.adisaputra.online"
RELAY_PORT="8443"

echo "Testing connection to relay server..."
echo "Host: $RELAY_HOST"
echo "Port: $RELAY_PORT"
echo

# Test basic connectivity
echo "[1/4] Testing basic connectivity..."
if ping -c 2 $RELAY_HOST >/dev/null 2>&1; then
    echo "✅ PASS: Relay server is reachable"
else
    echo "❌ FAIL: Cannot ping relay server"
    echo "Check internet connection and server IP"
fi

echo
echo "[2/4] Testing HTTPS port..."
if command -v nc >/dev/null 2>&1; then
    if timeout 5 nc -z $RELAY_HOST $RELAY_PORT 2>/dev/null; then
        echo "✅ PASS: Port $RELAY_PORT is open"
    else
        echo "❌ FAIL: Port $RELAY_PORT is closed or filtered"
    fi
elif command -v telnet >/dev/null 2>&1; then
    if echo | timeout 5 telnet $RELAY_HOST $RELAY_PORT 2>/dev/null | grep -q "Connected"; then
        echo "✅ PASS: Port $RELAY_PORT is open"
    else
        echo "❌ FAIL: Port $RELAY_PORT is closed or filtered"
    fi
else
    echo "⚠️  SKIP: Neither nc nor telnet available to test port"
fi

echo
echo "[3/4] Testing WebSocket endpoint..."
if command -v curl >/dev/null 2>&1; then
    if curl -k -s --max-time 10 "https://$RELAY_HOST/health" >/dev/null 2>&1; then
        echo "✅ PASS: Relay server health endpoint responding"
    else
        echo "❌ FAIL: Cannot reach relay server health endpoint"
    fi
elif command -v wget >/dev/null 2>&1; then
    if wget --no-check-certificate -q --timeout=10 -O /dev/null "https://$RELAY_HOST/health" 2>/dev/null; then
        echo "✅ PASS: Relay server health endpoint responding"
    else
        echo "❌ FAIL: Cannot reach relay server health endpoint"
    fi
else
    echo "⚠️  SKIP: Neither curl nor wget available to test HTTP"
fi

echo
echo "[4/4] Testing local agent binary..."
if [ -f "bin/agent" ]; then
    echo "✅ PASS: Agent binary found"
elif [ -f "bin/agent-linux" ]; then
    echo "✅ PASS: Agent Linux binary found"
else
    echo "❌ FAIL: Agent binary not found - run build.sh or make build first"
fi

echo
echo "========================================"
echo "Test Results Summary:"
echo "========================================"
echo
echo "If all tests pass, you can:"
echo "1. On server 103.195.169.32: Run the relay server"
echo "2. On your laptop: Run start-agent script"
echo "3. From remote location: Run client"
echo
echo "Next steps:"
echo "- Edit .env.production with your secure token"
echo "- Run agent on this machine"
echo "- Configure relay server on 103.195.169.32"
echo
