#!/bin/bash
# Demo script for Remote Tunnel on Linux/macOS

set -e

echo "Starting Remote Tunnel Demo on Linux..."
echo

# Set common token
export TUNNEL_TOKEN="demo-secret-token"

# Check if binaries exist
if [ ! -f "bin/relay" ]; then
    echo "Error: relay binary not found. Please run build.sh first."
    exit 1
fi

if [ ! -f "bin/agent" ]; then
    echo "Error: agent binary not found. Please run build.sh first."
    exit 1
fi

if [ ! -f "bin/client" ]; then
    echo "Error: client binary not found. Please run build.sh first."
    exit 1
fi

echo "Token set to: $TUNNEL_TOKEN"
echo

echo "Instructions:"
echo "1. This script will start all components in background"
echo "2. Check logs in /tmp/tunnel-*.log"
echo "3. Test with: ssh -p 2222 localhost (if SSH server is running)"
echo "4. Stop with: pkill -f 'relay|agent|client'"
echo

# Start relay server in background
echo "Starting relay server on port 8443..."
./bin/relay -addr :8443 -token "$TUNNEL_TOKEN" > /tmp/tunnel-relay.log 2>&1 &
RELAY_PID=$!

# Wait for relay to start
sleep 3

# Start agent in background
echo "Starting agent..."
./bin/agent -id demo-agent -relay-url wss://localhost:8443/ws/agent -allow 127.0.0.1: -token "$TUNNEL_TOKEN" > /tmp/tunnel-agent.log 2>&1 &
AGENT_PID=$!

# Wait for agent to connect
sleep 3

# Start client in background
echo "Starting client on port 2222..."
./bin/client -L :2222 -relay-url wss://localhost:8443/ws/client -agent demo-agent -target 127.0.0.1:22 -token "$TUNNEL_TOKEN" > /tmp/tunnel-client.log 2>&1 &
CLIENT_PID=$!

echo
echo "Demo started! All components running in background."
echo "PIDs: Relay=$RELAY_PID, Agent=$AGENT_PID, Client=$CLIENT_PID"
echo
echo "Logs:"
echo "- Relay: tail -f /tmp/tunnel-relay.log"
echo "- Agent: tail -f /tmp/tunnel-agent.log"  
echo "- Client: tail -f /tmp/tunnel-client.log"
echo
echo "To test the tunnel:"
echo "  ssh -p 2222 localhost"
echo
echo "To stop all components:"
echo "  kill $RELAY_PID $AGENT_PID $CLIENT_PID"
echo "  # Or: pkill -f 'relay|agent|client'"
echo

# Function to cleanup on exit
cleanup() {
    echo "Cleaning up..."
    kill $RELAY_PID $AGENT_PID $CLIENT_PID 2>/dev/null || true
    exit 0
}

# Trap signals for cleanup
trap cleanup INT TERM

echo "Press Ctrl+C to stop all components..."
wait
