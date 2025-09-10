#!/bin/bash

# Stop tunnel server on Linux
echo "🛑 Stopping Tunnel Server..."

# Check if PID file exists
if [ -f "server.pid" ]; then
    PID=$(cat server.pid)
    
    if kill -0 $PID 2>/dev/null; then
        echo "📍 Found server process with PID: $PID"
        
        # Send SIGTERM first
        echo "🔄 Sending SIGTERM..."
        kill -TERM $PID
        
        # Wait up to 10 seconds for graceful shutdown
        for i in {1..10}; do
            if ! kill -0 $PID 2>/dev/null; then
                echo "✅ Server stopped gracefully"
                rm -f server.pid
                exit 0
            fi
            sleep 1
        done
        
        # If still running, force kill
        echo "⚠️ Server didn't stop gracefully, force killing..."
        kill -KILL $PID 2>/dev/null
        
        if ! kill -0 $PID 2>/dev/null; then
            echo "✅ Server force stopped"
            rm -f server.pid
        else
            echo "❌ Failed to stop server"
            exit 1
        fi
    else
        echo "⚠️ PID file exists but process not running"
        rm -f server.pid
    fi
else
    echo "📍 No PID file found, checking for running processes..."
fi

# Check for any remaining tunnel-server processes
PIDS=$(pgrep -f "tunnel-server")
if [ ! -z "$PIDS" ]; then
    echo "📍 Found tunnel-server processes: $PIDS"
    
    for pid in $PIDS; do
        echo "🔄 Stopping process $pid..."
        kill -TERM $pid 2>/dev/null
        
        # Wait 5 seconds
        sleep 5
        
        if kill -0 $pid 2>/dev/null; then
            echo "⚠️ Force killing process $pid..."
            kill -KILL $pid 2>/dev/null
        fi
    done
    
    echo "✅ All tunnel-server processes stopped"
else
    echo "ℹ️ No tunnel-server processes found"
fi

echo "🏁 Server stop operation completed"
