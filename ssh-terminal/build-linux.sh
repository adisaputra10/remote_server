#!/bin/bash

# Build script for Linux deployment
echo "🔨 Building tunnel system for Linux..."

# Set Go environment for cross-compilation
export GOOS=linux
export GOARCH=amd64

# Create build directory
mkdir -p build/linux

# Build server
echo "📦 Building server..."
go build -o build/linux/tunnel-server ./cmd/server
if [ $? -eq 0 ]; then
    echo "✅ Server built successfully"
else
    echo "❌ Server build failed"
    exit 1
fi

# Build agent
echo "📦 Building agent..."
go build -o build/linux/tunnel-agent ./cmd/agent
if [ $? -eq 0 ]; then
    echo "✅ Agent built successfully"
else
    echo "❌ Agent build failed"
    exit 1
fi

# Build client
echo "📦 Building client..."
go build -o build/linux/tunnel-client ./cmd/client
if [ $? -eq 0 ]; then
    echo "✅ Client built successfully"
else
    echo "❌ Client build failed"
    exit 1
fi

# Copy configuration files
echo "📋 Copying configuration files..."
cp agent-config-db.json build/linux/ 2>/dev/null || echo "⚠️ agent-config-db.json not found"
cp server-config-db.json build/linux/ 2>/dev/null || echo "⚠️ server-config-db.json not found"
cp client-config-clean.json build/linux/ 2>/dev/null || echo "⚠️ client-config-clean.json not found"

# Make binaries executable
chmod +x build/linux/tunnel-*

echo "🎉 Build completed! Files are in build/linux/"
echo ""
echo "To deploy to Linux server:"
echo "1. Copy build/linux/* to your Linux server"
echo "2. Run: ./start-server-linux.sh"
echo "3. Run: ./start-agent-linux.sh"
