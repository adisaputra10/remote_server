#!/bin/bash

# Start tunnel server on Linux
echo "🚀 Starting Tunnel Server..."

# Default configuration
SERVER_PORT=${SERVER_PORT:-8080}
SERVER_HOST=${SERVER_HOST:-"0.0.0.0"}
CONFIG_FILE=${CONFIG_FILE:-"server-config-db.json"}
LOG_FILE=${LOG_FILE:-"server.log"}

# Check if binary exists
if [ ! -f "./tunnel-server" ]; then
    echo "❌ tunnel-server binary not found!"
    echo "Please run build-linux.sh first or copy the binary to this directory"
    exit 1
fi

# Create default config if not exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "📋 Creating default server configuration..."
    cat > "$CONFIG_FILE" << EOF
{
  "port": $SERVER_PORT,
  "host": "$SERVER_HOST",
  "log_file": "$LOG_FILE",
  "token": "your-secure-token-here",
  "database_url": "",
  "enable_database": false,
  "tls_cert": "",
  "tls_key": ""
}
EOF
    echo "✅ Created $CONFIG_FILE"
fi

# Create logs directory
mkdir -p logs

# Check if server is already running
if pgrep -f "tunnel-server" > /dev/null; then
    echo "⚠️ Server is already running!"
    echo "Current processes:"
    pgrep -f "tunnel-server" -l
    echo ""
    echo "To stop existing server, run: ./stop-server-linux.sh"
    exit 1
fi

# Start server
echo "🔧 Configuration:"
echo "  - Port: $SERVER_PORT"
echo "  - Host: $SERVER_HOST"
echo "  - Config: $CONFIG_FILE"
echo "  - Log: $LOG_FILE"
echo ""

# Run server in background
nohup ./tunnel-server \
    -config="$CONFIG_FILE" \
    -port=$SERVER_PORT \
    -host="$SERVER_HOST" \
    > logs/server-startup.log 2>&1 &

SERVER_PID=$!
echo "🚀 Server started with PID: $SERVER_PID"

# Wait a moment and check if it's still running
sleep 2
if kill -0 $SERVER_PID 2>/dev/null; then
    echo "✅ Server is running successfully!"
    echo "📊 Health check: http://$SERVER_HOST:$SERVER_PORT/health"
    echo "🔌 Agent endpoint: ws://$SERVER_HOST:$SERVER_PORT/agent"
    echo "👤 Client endpoint: ws://$SERVER_HOST:$SERVER_PORT/client"
    echo ""
    echo "To view logs: tail -f $LOG_FILE"
    echo "To stop server: ./stop-server-linux.sh"
    
    # Save PID for stop script
    echo $SERVER_PID > server.pid
else
    echo "❌ Server failed to start!"
    echo "Check logs:"
    cat logs/server-startup.log
    exit 1
fi
