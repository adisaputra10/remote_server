#!/bin/bash
# Stop relay server safely

echo "========================================"
echo "Stopping Remote Tunnel Relay Server"
echo "========================================"

echo "Checking for running relay processes..."

# Find relay processes
RELAY_PIDS=$(pgrep -f "bin/relay" 2>/dev/null)

if [ -n "$RELAY_PIDS" ]; then
    echo "Found relay processes: $RELAY_PIDS"
    
    for PID in $RELAY_PIDS; do
        echo "Stopping relay process $PID..."
        
        # Get process info
        ps -p $PID -o pid,ppid,cmd --no-headers 2>/dev/null || echo "Process $PID not found"
        
        # Send SIGTERM first (graceful shutdown)
        kill -TERM $PID 2>/dev/null
        
        # Wait up to 10 seconds for graceful shutdown
        for i in {1..10}; do
            if ! kill -0 $PID 2>/dev/null; then
                echo "✅ Process $PID stopped gracefully"
                break
            fi
            sleep 1
        done
        
        # Force kill if still running
        if kill -0 $PID 2>/dev/null; then
            echo "⚠️  Force killing process $PID"
            kill -KILL $PID 2>/dev/null
            sleep 1
        fi
    done
else
    echo "ℹ️  No relay processes found"
fi

# Check for processes still listening on port 443 and 8443
echo
echo "Checking ports..."

PORT_443=$(netstat -tlnp 2>/dev/null | grep ":443 " | grep LISTEN)
PORT_8443=$(netstat -tlnp 2>/dev/null | grep ":8443 " | grep LISTEN)

if [ -n "$PORT_443" ]; then
    echo "⚠️  Port 443 still in use:"
    echo "$PORT_443"
    
    # Extract PID and kill
    PID_443=$(echo "$PORT_443" | awk '{print $7}' | cut -d'/' -f1)
    if [ -n "$PID_443" ] && [ "$PID_443" != "-" ]; then
        echo "Killing process $PID_443 on port 443..."
        kill -TERM "$PID_443" 2>/dev/null
        sleep 2
        kill -KILL "$PID_443" 2>/dev/null
    fi
fi

if [ -n "$PORT_8443" ]; then
    echo "⚠️  Port 8443 still in use:"
    echo "$PORT_8443"
fi

# Final check
sleep 2
echo
echo "Final port check..."
netstat -tlnp 2>/dev/null | grep -E ":(443|8443) " | grep LISTEN || echo "✅ Ports 443 and 8443 are now free"

echo
echo "========================================"
echo "Relay Server Stopped"
echo "========================================"
echo
echo "To start relay on port 8443:"
echo "./start-relay.sh"
echo
echo "To check ports:"
echo "netstat -tlnp | grep -E ':(443|8443) '"
echo
