#!/bin/bash
# Production setup script for Linux laptop (Agent)
# Relay Server: 103.195.169.32

set -e

echo "========================================"
echo "Remote Tunnel - Laptop Agent Setup"
echo "========================================"
echo "Relay Server: sh.adisaputra.online"
echo "Agent: Local Laptop"
echo "========================================"

# Load configuration
if [ -f ".env.production" ]; then
    echo "Loading production configuration..."
    export $(grep -v '^#' .env.production | xargs)
else
    echo "Warning: .env.production not found, using defaults"
    export TUNNEL_TOKEN="change-this-token"
    export AGENT_ID="laptop-agent"
    export AGENT_RELAY_URL="wss://sh.adisaputra.online:8443/ws/agent"
fi

echo
echo "Configuration:"
echo "- Token: $TUNNEL_TOKEN"
echo "- Agent ID: $AGENT_ID"
echo "- Relay URL: $AGENT_RELAY_URL"
echo

# Check if binaries exist
AGENT_BINARY=""
if [ -f "bin/agent" ]; then
    AGENT_BINARY="bin/agent"
elif [ -f "bin/agent-linux" ]; then
    AGENT_BINARY="bin/agent-linux"
else
    echo "Error: agent binary not found. Building..."
    make build
    if [ $? -ne 0 ]; then
        echo "Build failed!"
        exit 1
    fi
    AGENT_BINARY="bin/agent"
fi

echo "Using binary: $AGENT_BINARY"
echo

echo "Services to expose:"
echo "[1] SSH Server (port 22)"
echo "[2] Web Server (port 8080)"
echo "[3] Database (port 5432)"
echo "[4] Custom ports"
echo "[5] All common services"
read -p "Select option (1-5): " choice

case $choice in
    1)
        ALLOW_PORTS="-allow 127.0.0.1:22"
        echo "Selected: SSH Server only"
        ;;
    2)
        ALLOW_PORTS="-allow 127.0.0.1:8080"
        echo "Selected: Web Server only"
        ;;
    3)
        ALLOW_PORTS="-allow 127.0.0.1:5432"
        echo "Selected: Database only"
        ;;
    4)
        read -p "Enter ports (e.g., 127.0.0.1:3000 127.0.0.1:8000): " custom_ports
        ALLOW_PORTS="-allow ${custom_ports// / -allow }"
        echo "Selected: Custom ports"
        ;;
    *)
        ALLOW_PORTS="-allow 127.0.0.1:22 -allow 127.0.0.1:80 -allow 127.0.0.1:443 -allow 127.0.0.1:3000 -allow 127.0.0.1:8080 -allow 127.0.0.1:5432"
        echo "Selected: All common services"
        ;;
esac

echo
echo "Starting agent with configuration:"
echo "$AGENT_BINARY -id $AGENT_ID -relay-url $AGENT_RELAY_URL $ALLOW_PORTS -token $TUNNEL_TOKEN"
echo
echo "Press Ctrl+C to stop the agent"
echo "========================================"

# Start agent
exec $AGENT_BINARY -id "$AGENT_ID" -relay-url "$AGENT_RELAY_URL" $ALLOW_PORTS -token "$TUNNEL_TOKEN"
