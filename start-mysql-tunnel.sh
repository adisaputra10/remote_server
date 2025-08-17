#!/bin/bash
# MySQL/MariaDB tunnel script
# This script creates a tunnel specifically for MySQL/MariaDB access

echo "========================================"
echo "MySQL/MariaDB Remote Tunnel"
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
fi

echo
echo "Available agents to connect to:"
echo "- $AGENT_ID (your laptop)"
echo
read -p "Enter agent ID [$AGENT_ID]: " target_agent
target_agent=${target_agent:-$AGENT_ID}

echo
echo "MySQL/MariaDB Configuration:"
echo "[1] Default MySQL/MariaDB (localhost:3306)"
echo "[2] Custom MySQL/MariaDB server"
read -p "Select option (1-2): " mysql_choice

case $mysql_choice in
    1)
        TARGET_ADDR="127.0.0.1:3306"
        LOCAL_PORT="3306"
        echo "Selected: Default MySQL/MariaDB on localhost:3306"
        ;;
    *)
        read -p "Enter MySQL server address (e.g., 192.168.1.100:3306): " TARGET_ADDR
        read -p "Enter local port [3306]: " LOCAL_PORT
        LOCAL_PORT=${LOCAL_PORT:-3306}
        echo "Selected: Custom MySQL/MariaDB server $TARGET_ADDR"
        ;;
esac

echo
echo "========================================"
echo "Starting MySQL/MariaDB tunnel:"
echo "Local port $LOCAL_PORT -> Agent $target_agent -> MySQL $TARGET_ADDR"
echo
echo "After connection established, you can connect using:"
echo "  mysql -h localhost -P $LOCAL_PORT -u your_username -p"
echo "  or use any MySQL client with host: localhost, port: $LOCAL_PORT"
echo
echo "MySQL Workbench connection:"
echo "  Host: localhost"
echo "  Port: $LOCAL_PORT"
echo "  Username: your_mysql_username"
echo
echo "Press Ctrl+C to stop the tunnel"
echo "========================================"
echo

./bin/client -L :$LOCAL_PORT -relay-url wss://sh.adisaputra.online:8443/ws/client -agent $target_agent -target $TARGET_ADDR -token $TUNNEL_TOKEN -insecure

echo
echo "MySQL/MariaDB tunnel stopped."
read -p "Press Enter to continue..."
