#!/bin/bash
# Client script for testing connection through relay
# Use this to test from another machine

echo "========================================"
echo "Remote Tunnel - Test Client"
echo "========================================"
echo "Relay Server: sh.adisaputra.online"
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
    CLIENT_RELAY_URL="wss://sh.adisaputra.online:8443/ws/client"
fi

echo
echo "Available agents to connect to:"
echo "- $AGENT_ID (your laptop)"
echo
read -p "Enter agent ID [$AGENT_ID]: " target_agent
target_agent=${target_agent:-$AGENT_ID}

echo
echo "Common target services:"
echo "[1] SSH (port 22)"
echo "[2] Web Server (port 8080)"
echo "[3] PostgreSQL (port 5432)"
echo "[4] MySQL/MariaDB (port 3306)"
echo "[5] Custom"
read -p "Select service (1-5): " service_choice

case $service_choice in
    1)
        TARGET_ADDR="127.0.0.1:22"
        LOCAL_PORT="2222"
        echo "Selected: SSH - Access via ssh -p 2222 user@localhost"
        ;;
    2)
        TARGET_ADDR="127.0.0.1:8080"
        LOCAL_PORT="8080"
        echo "Selected: Web Server - Access via http://localhost:8080"
        ;;
    3)
        TARGET_ADDR="127.0.0.1:5432"
        LOCAL_PORT="5432"
        echo "Selected: PostgreSQL - Access via localhost:5432"
        ;;
    4)
        TARGET_ADDR="127.0.0.1:3306"
        LOCAL_PORT="3306"
        echo "Selected: MySQL/MariaDB - Access via localhost:3306"
        echo "Example: mysql -h localhost -P 3306 -u username -p"
        ;;
    *)
        read -p "Enter target address (e.g., 127.0.0.1:3000): " TARGET_ADDR
        read -p "Enter local port (e.g., 3000): " LOCAL_PORT
        echo "Selected: Custom service"
        ;;
esac

echo
echo "Starting client tunnel:"
echo "Local port $LOCAL_PORT -> Agent $target_agent -> Target $TARGET_ADDR"
echo
echo "Command: bin/client -L :$LOCAL_PORT -relay-url wss://sh.adisaputra.online:8443/ws/client -agent $target_agent -target $TARGET_ADDR -token $TUNNEL_TOKEN -insecure"
echo
echo "Press Ctrl+C to stop"
echo "========================================"

bin/client -L :$LOCAL_PORT -relay-url wss://sh.adisaputra.online:8443/ws/client -agent $target_agent -target $TARGET_ADDR -token $TUNNEL_TOKEN -insecure

echo
echo "Client stopped."
read -p "Press Enter to continue..."
