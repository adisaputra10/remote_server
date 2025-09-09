#!/bin/bash

# Start GoTeleport Server
echo "🚀 Starting GoTeleport Server..."

# Check if config exists
if [ ! -f "server-config-db.json" ]; then
    echo "❌ server-config-db.json not found"
    exit 1
fi

# Check if binary exists
if [ ! -f "goteleport-server-db" ]; then
    echo "❌ goteleport-server-db binary not found"
    exit 1
fi

# Create logs directory
mkdir -p logs

# Stop existing server if running
pkill -f "goteleport-server-db" 2>/dev/null

# Start server in background
echo "Starting server on port 8081..."
nohup ./goteleport-server-db server-config-db.json > logs/server.log 2>&1 &
SERVER_PID=$!

# Wait a moment for startup
sleep 2

# Check if server started successfully
if ps -p $SERVER_PID > /dev/null; then
    echo "✅ GoTeleport Server started successfully (PID: $SERVER_PID)"
    echo "📝 Log file: logs/server.log"
    echo "🔗 WebSocket endpoint: ws://$(hostname -I | awk '{print $1}'):8081/ws/client"
    echo "🔗 Tunnel endpoint: ws://$(hostname -I | awk '{print $1}'):8081/ws/tunnel"
    
    # Save PID for later management
    echo $SERVER_PID > logs/server.pid
    
    echo ""
    echo "📊 Server status:"
    echo "   PID: $SERVER_PID"
    echo "   Ports: 8081 (WebSocket), 8082 (API)"
    echo "   Config: server-config-db.json"
    echo "   Logs: logs/server.log"
    
    echo ""
    echo "🔧 Management commands:"
    echo "   View logs: tail -f logs/server.log"
    echo "   Stop server: kill $SERVER_PID"
    echo "   Check status: ps -p $SERVER_PID"
else
    echo "❌ Failed to start GoTeleport Server"
    echo "📝 Check logs/server.log for details"
    exit 1
fi
