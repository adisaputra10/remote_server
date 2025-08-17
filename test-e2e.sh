#!/bin/bash
# End-to-end test script for Linux

set -e

echo "Running Remote Tunnel E2E Tests..."

# Configuration
TOKEN="test-token-$(date +%s)"
RELAY_PORT=8443
CLIENT_PORT=2224
AGENT_ID="test-agent"

# Cleanup function
cleanup() {
    echo "Cleaning up test processes..."
    pkill -f "test-relay|test-agent|test-client" 2>/dev/null || true
    rm -f /tmp/test-server.crt /tmp/test-server.key
    exit 0
}

trap cleanup INT TERM EXIT

# Check if binaries exist
if [ ! -f "bin/relay" ] || [ ! -f "bin/agent" ] || [ ! -f "bin/client" ]; then
    echo "Binaries not found. Building..."
    make build
fi

echo "Starting test components..."

# Start relay server
echo "Starting relay server on port $RELAY_PORT..."
export TUNNEL_TOKEN="$TOKEN"
./bin/relay -addr ":$RELAY_PORT" -cert /tmp/test-server.crt -key /tmp/test-server.key -token "$TOKEN" > /tmp/test-relay.log 2>&1 &
RELAY_PID=$!

# Wait for relay to start and generate certificates
sleep 5

# Start a simple test server (netcat)
echo "Starting test server on port 8888..."
{ echo -e "HTTP/1.1 200 OK\r\nContent-Length: 13\r\n\r\nHello, World!"; } | nc -l -p 8888 > /dev/null 2>&1 &
TEST_SERVER_PID=$!

# Start agent
echo "Starting agent..."
./bin/agent -id "$AGENT_ID" -relay-url "wss://localhost:$RELAY_PORT/ws/agent" -allow 127.0.0.1:8888 -token "$TOKEN" > /tmp/test-agent.log 2>&1 &
AGENT_PID=$!

# Wait for agent to connect
sleep 3

# Start client
echo "Starting client on port $CLIENT_PORT..."
./bin/client -L ":$CLIENT_PORT" -relay-url "wss://localhost:$RELAY_PORT/ws/client" -agent "$AGENT_ID" -target 127.0.0.1:8888 -token "$TOKEN" > /tmp/test-client.log 2>&1 &
CLIENT_PID=$!

# Wait for client to start
sleep 3

# Test the tunnel
echo "Testing tunnel..."
RESPONSE=$(curl -s --max-time 10 "http://localhost:$CLIENT_PORT" 2>/dev/null || echo "FAILED")

if [ "$RESPONSE" = "Hello, World!" ]; then
    echo "✅ SUCCESS: Tunnel working correctly!"
    echo "Response: $RESPONSE"
else
    echo "❌ FAILED: Tunnel not working"
    echo "Response: $RESPONSE"
    echo
    echo "Logs:"
    echo "=== Relay Log ==="
    cat /tmp/test-relay.log
    echo "=== Agent Log ==="
    cat /tmp/test-agent.log
    echo "=== Client Log ==="
    cat /tmp/test-client.log
    exit 1
fi

echo
echo "Test completed successfully!"
echo "Cleaning up..."

# Cleanup will be called by trap
