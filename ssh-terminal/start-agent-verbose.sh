#!/bin/bash

echo "🚀 Starting GoTeleport Agent with Verbose Logging..."

# Build agent
echo "📦 Building agent..."
go build -o goteleport-agent-db goteleport-agent-db.go

if [ $? -ne 0 ]; then
    echo "❌ Failed to build agent"
    exit 1
fi

echo "✅ Agent built successfully"

# Check if config exists
if [ ! -f "agent-config-db.json" ]; then
    echo "❌ agent-config-db.json not found"
    exit 1
fi

echo "📋 Starting agent with config:"
cat agent-config-db.json | jq .

echo ""
echo "🔧 Starting agent with verbose output..."
echo "📝 Logs will be written to both agent-db.log and stdout"
echo ""

# Start agent
./goteleport-agent-db agent-config-db.json
