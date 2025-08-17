#!/bin/bash
# MySQL/MariaDB Agent script
# This script starts agent specifically configured for MySQL/MariaDB access

echo "========================================"
echo "MySQL/MariaDB Agent Setup"
echo "========================================"
echo "Relay Server: sh.adisaputra.online"
echo "Agent: Local Laptop (MySQL/MariaDB enabled)"
echo "========================================"

# Load configuration
if [ -f .env.production ]; then
    echo "Loading production configuration..."
    set -a  # automatically export all variables
    source .env.production
    set +a
else
    echo "Warning: .env.production not found, using defaults"
    TUNNEL_TOKEN="change-this-token"
    AGENT_ID="laptop-agent"
    AGENT_RELAY_URL="wss://sh.adisaputra.online:8443/ws/agent"
fi

echo
echo "Configuration:"
echo "- Token: $TUNNEL_TOKEN"
echo "- Agent ID: $AGENT_ID"
echo "- Relay URL: $AGENT_RELAY_URL"
echo

# Check binary
if [ -f "bin/agent" ]; then
    AGENT_BINARY="bin/agent"
elif [ -f "./agent" ]; then
    AGENT_BINARY="./agent"
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

echo "MySQL/MariaDB Services Configuration:"
echo "[1] MySQL/MariaDB only (port 3306)"
echo "[2] MySQL + Web Server (ports 3306, 8080)"
echo "[3] MySQL + SSH (ports 3306, 22)"
echo "[4] All services including MySQL"
read -p "Select option (1-4): " choice

case $choice in
    1)
        ALLOW_PORTS="-allow 127.0.0.1:3306"
        echo "Selected: MySQL/MariaDB only (port 3306)"
        ;;
    2)
        ALLOW_PORTS="-allow 127.0.0.1:3306 -allow 127.0.0.1:8080"
        echo "Selected: MySQL/MariaDB + Web Server (ports 3306, 8080)"
        ;;
    3)
        ALLOW_PORTS="-allow 127.0.0.1:3306 -allow 127.0.0.1:22"
        echo "Selected: MySQL/MariaDB + SSH (ports 3306, 22)"
        ;;
    *)
        ALLOW_PORTS="-allow 127.0.0.1:3306 -allow 127.0.0.1:22 -allow 127.0.0.1:80 -allow 127.0.0.1:443 -allow 127.0.0.1:8080 -allow 127.0.0.1:5432"
        echo "Selected: All services including MySQL/MariaDB"
        ;;
esac

echo
echo "========================================"
echo "Starting MySQL/MariaDB agent:"
echo
echo "Make sure MySQL/MariaDB is running and accessible on localhost:3306"
echo
echo "Command: $AGENT_BINARY -id $AGENT_ID -relay-url $AGENT_RELAY_URL $ALLOW_PORTS -token $TUNNEL_TOKEN -insecure"
echo
echo "Press Ctrl+C to stop the agent"
echo "========================================"

# Start agent with insecure flag for self-signed certificates
exec $AGENT_BINARY -id "$AGENT_ID" -relay-url "$AGENT_RELAY_URL" $ALLOW_PORTS -token "$TUNNEL_TOKEN" -insecure
