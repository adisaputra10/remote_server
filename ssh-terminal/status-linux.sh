#!/bin/bash

# Status check for tunnel system on Linux
echo "📊 Tunnel System Status Check"
echo "=============================="

# Check server status
echo "🖥️ SERVER STATUS:"
if [ -f "server.pid" ]; then
    PID=$(cat server.pid)
    if kill -0 $PID 2>/dev/null; then
        echo "✅ Server is running (PID: $PID)"
        
        # Check if port is listening
        SERVER_PORT=8080
        if netstat -tuln 2>/dev/null | grep -q ":$SERVER_PORT "; then
            echo "✅ Server is listening on port $SERVER_PORT"
        else
            echo "⚠️ Server process running but port $SERVER_PORT not listening"
        fi
        
        # Try health check
        if command -v curl >/dev/null 2>&1; then
            echo "🔍 Health check..."
            if curl -s "http://localhost:$SERVER_PORT/health" >/dev/null; then
                echo "✅ Server health check passed"
            else
                echo "⚠️ Server health check failed"
            fi
        fi
    else
        echo "❌ Server PID file exists but process not running"
        rm -f server.pid
    fi
else
    echo "❌ Server is not running (no PID file)"
fi

echo ""

# Check agent status
echo "🤖 AGENT STATUS:"
if [ -f "agent.pid" ]; then
    PID=$(cat agent.pid)
    if kill -0 $PID 2>/dev/null; then
        echo "✅ Agent is running (PID: $PID)"
        
        # Check recent logs for connection status
        if [ -f "agent.log" ]; then
            if tail -n 20 agent.log | grep -q "Connected\|Heartbeat"; then
                echo "✅ Agent appears to be connected"
            else
                echo "⚠️ Agent may not be connected (check logs)"
            fi
        fi
    else
        echo "❌ Agent PID file exists but process not running"
        rm -f agent.pid
    fi
else
    echo "❌ Agent is not running (no PID file)"
fi

echo ""

# Check for any orphaned processes
echo "🔍 PROCESS CHECK:"
SERVER_PROCS=$(pgrep -f "tunnel-server" | wc -l)
AGENT_PROCS=$(pgrep -f "tunnel-agent" | wc -l)

echo "Server processes: $SERVER_PROCS"
echo "Agent processes: $AGENT_PROCS"

if [ $SERVER_PROCS -gt 1 ]; then
    echo "⚠️ Multiple server processes detected"
    pgrep -f "tunnel-server" -l
fi

if [ $AGENT_PROCS -gt 1 ]; then
    echo "⚠️ Multiple agent processes detected"
    pgrep -f "tunnel-agent" -l
fi

echo ""

# Check log files
echo "📝 LOG FILES:"
for logfile in server.log agent.log logs/server-startup.log logs/agent-startup.log; do
    if [ -f "$logfile" ]; then
        size=$(stat -f%z "$logfile" 2>/dev/null || stat -c%s "$logfile" 2>/dev/null || echo "unknown")
        echo "✅ $logfile (${size} bytes)"
    else
        echo "❌ $logfile (not found)"
    fi
done

echo ""

# Network status
echo "🌐 NETWORK STATUS:"
if command -v netstat >/dev/null 2>&1; then
    echo "Listening ports:"
    netstat -tuln | grep -E ":8080|:3306|:5432" || echo "No relevant ports found"
fi

echo ""

# Quick help
echo "📚 QUICK COMMANDS:"
echo "Start server: ./start-server-linux.sh"
echo "Start agent:  ./start-agent-linux.sh" 
echo "Stop server:  ./stop-server-linux.sh"
echo "Stop agent:   ./stop-agent-linux.sh"
echo "View server logs: tail -f server.log"
echo "View agent logs:  tail -f agent.log"
