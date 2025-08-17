#!/bin/bash
# Cleanup script for remote tunnel

echo "========================================"
echo "Remote Tunnel - Cleanup"
echo "========================================"

echo "Stopping all tunnel processes..."

# Kill agent processes
if pgrep -f "bin/agent" >/dev/null; then
    echo "Stopping agent processes..."
    pkill -f "bin/agent"
    echo "✅ Agent processes stopped"
else
    echo "ℹ️  No agent processes running"
fi

# Kill client processes
if pgrep -f "bin/client" >/dev/null; then
    echo "Stopping client processes..."
    pkill -f "bin/client"
    echo "✅ Client processes stopped"
else
    echo "ℹ️  No client processes running"
fi

# Kill relay processes
if pgrep -f "bin/relay" >/dev/null; then
    echo "Stopping relay processes..."
    pkill -f "bin/relay"
    echo "✅ Relay processes stopped"
else
    echo "ℹ️  No relay processes running"
fi

echo
echo "Cleaning up log files..."

# Clean log files
if ls *.log >/dev/null 2>&1; then
    rm -f *.log
    echo "✅ Log files cleaned"
else
    echo "ℹ️  No log files to clean"
fi

if [ -d "logs" ] && ls logs/*.log >/dev/null 2>&1; then
    rm -f logs/*.log
    echo "✅ Logs directory cleaned"
fi

# Clean temp files
if [ -d "tmp" ]; then
    rm -rf tmp
    echo "✅ Temp directory cleaned"
fi

echo
echo "Cleaning up build artifacts..."

# Clean Go build cache
echo "Cleaning Go build cache..."
go clean -cache >/dev/null 2>&1
go clean -modcache >/dev/null 2>&1
echo "✅ Go cache cleaned"

echo
echo "========================================"
echo "Cleanup Complete!"
echo "========================================"
echo
echo "What was cleaned:"
echo "- All running tunnel processes (agent, client, relay)"
echo "- Log files (*.log, logs/*.log)"
echo "- Temporary files and directories"
echo "- Go build cache"
echo
echo "What was preserved:"
echo "- Configuration files (.env.production, config/*.yaml)"
echo "- Certificates (certs/*)"
echo "- Built binaries (bin/*)"
echo "- Source code"
echo
echo "To start fresh:"
echo "1. Run ./build.sh to rebuild binaries"
echo "2. Run ./setup-domain.sh for domain configuration"
echo "3. Run ./start-agent.sh to begin tunneling"
echo

read -p "Press Enter to continue..."
