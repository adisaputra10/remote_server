#!/bin/bash

# Start tunnel agent on Linux
echo "🤖 Starting Tunnel Agent..."

# Default configuration
SERVER_URL=${SERVER_URL:-"ws://localhost:8080"}
CONFIG_FILE=${CONFIG_FILE:-"agent-config-db.json"}
AGENT_NAME=${AGENT_NAME:-"linux-agent-$(hostname)"}

# Check if binary exists
if [ ! -f "./tunnel-agent" ]; then
    echo "❌ tunnel-agent binary not found!"
    echo "Please run build-linux.sh first or copy the binary to this directory"
    exit 1
fi

# Create default config if not exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "📋 Creating default agent configuration..."
    cat > "$CONFIG_FILE" << EOF
{
  "server_url": "$SERVER_URL",
  "agent_name": "$AGENT_NAME",
  "token": "your-secure-token-here",
  "reconnect_interval": 30,
  "heartbeat_interval": 30,
  "log_file": "agent.log",
  "database_proxy": {
    "enabled": true,
    "mysql_port": 3306,
    "postgres_port": 5432,
    "log_commands": true
  },
  "capabilities": [
    "tcp_tunnel",
    "database_proxy",
    "port_forward"
  ]
}
EOF
    echo "✅ Created $CONFIG_FILE"
    echo "⚠️ Please edit $CONFIG_FILE to set correct server_url and token"
fi

# Create logs directory
mkdir -p logs

# Check if agent is already running
if pgrep -f "tunnel-agent" > /dev/null; then
    echo "⚠️ Agent is already running!"
    echo "Current processes:"
    pgrep -f "tunnel-agent" -l
    echo ""
    echo "To stop existing agent, run: ./stop-agent-linux.sh"
    exit 1
fi

# Get system information
PLATFORM=$(uname -s)
ARCH=$(uname -m)
HOSTNAME=$(hostname)
IP_ADDRESS=$(hostname -I | awk '{print $1}')

echo "🔧 System Information:"
echo "  - Platform: $PLATFORM"
echo "  - Architecture: $ARCH"
echo "  - Hostname: $HOSTNAME"
echo "  - IP Address: $IP_ADDRESS"
echo "  - Agent Name: $AGENT_NAME"
echo "  - Server URL: $SERVER_URL"
echo "  - Config: $CONFIG_FILE"
echo ""

# Run agent in background
nohup ./tunnel-agent \
    -config="$CONFIG_FILE" \
    -server="$SERVER_URL" \
    -name="$AGENT_NAME" \
    > logs/agent-startup.log 2>&1 &

AGENT_PID=$!
echo "🤖 Agent started with PID: $AGENT_PID"

# Wait a moment and check if it's still running
sleep 2
if kill -0 $AGENT_PID 2>/dev/null; then
    echo "✅ Agent is running successfully!"
    echo "🔗 Attempting to connect to server..."
    echo ""
    echo "To view logs: tail -f agent.log"
    echo "To stop agent: ./stop-agent-linux.sh"
    
    # Save PID for stop script
    echo $AGENT_PID > agent.pid
    
    # Check connection after a few seconds
    sleep 3
    if grep -q "Connected to server" agent.log 2>/dev/null; then
        echo "🎉 Agent connected to server successfully!"
    else
        echo "⚠️ Agent may not be connected to server yet"
        echo "Check agent logs for connection status"
    fi
else
    echo "❌ Agent failed to start!"
    echo "Check logs:"
    cat logs/agent-startup.log
    exit 1
fi
