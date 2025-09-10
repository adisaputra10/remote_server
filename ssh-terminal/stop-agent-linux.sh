#!/bin/bash

# Stop tunnel agent on Linux
echo "🛑 Stopping Tunnel Agent..."

# Check if PID file exists
if [ -f "agent.pid" ]; then
    PID=$(cat agent.pid)
    
    if kill -0 $PID 2>/dev/null; then
        echo "📍 Found agent process with PID: $PID"
        
        # Send SIGTERM first
        echo "🔄 Sending SIGTERM..."
        kill -TERM $PID
        
        # Wait up to 10 seconds for graceful shutdown
        for i in {1..10}; do
            if ! kill -0 $PID 2>/dev/null; then
                echo "✅ Agent stopped gracefully"
                rm -f agent.pid
                exit 0
            fi
            sleep 1
        done
        
        # If still running, force kill
        echo "⚠️ Agent didn't stop gracefully, force killing..."
        kill -KILL $PID 2>/dev/null
        
        if ! kill -0 $PID 2>/dev/null; then
            echo "✅ Agent force stopped"
            rm -f agent.pid
        else
            echo "❌ Failed to stop agent"
            exit 1
        fi
    else
        echo "⚠️ PID file exists but process not running"
        rm -f agent.pid
    fi
else
    echo "📍 No PID file found, checking for running processes..."
fi

# Check for any remaining tunnel-agent processes
PIDS=$(pgrep -f "tunnel-agent")
if [ ! -z "$PIDS" ]; then
    echo "📍 Found tunnel-agent processes: $PIDS"
    
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
    
    echo "✅ All tunnel-agent processes stopped"
else
    echo "ℹ️ No tunnel-agent processes found"
fi

echo "🏁 Agent stop operation completed"
