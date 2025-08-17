#!/bin/bash
# Build script for Linux/macOS

set -e

echo "Building Remote Tunnel binaries for Linux..."

# Create bin directory
mkdir -p bin

# Download dependencies
echo "Downloading dependencies..."
go mod download
go mod tidy

# Build binaries for Linux
echo "Building relay..."
GOOS=linux GOARCH=amd64 go build -o bin/relay-linux ./cmd/relay

echo "Building agent..."
GOOS=linux GOARCH=amd64 go build -o bin/agent-linux ./cmd/agent

echo "Building client..."
GOOS=linux GOARCH=amd64 go build -o bin/client-linux ./cmd/client

# Also build for current platform (if running on Linux/Mac)
if [[ "$OSTYPE" == "linux-gnu"* ]] || [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Building for current platform..."
    go build -o bin/relay ./cmd/relay
    go build -o bin/agent ./cmd/agent
    go build -o bin/client ./cmd/client
    chmod +x bin/relay bin/agent bin/client
fi

chmod +x bin/relay-linux bin/agent-linux bin/client-linux

echo "Build complete!"
echo
echo "Linux binaries created in bin/ directory:"
echo "- relay-linux"
echo "- agent-linux"
echo "- client-linux"
echo
echo "To run a quick test on Linux:"
echo "1. Export token: export TUNNEL_TOKEN=test-token"
echo "2. Run relay: ./bin/relay-linux -addr :8443"
echo "3. Run agent: ./bin/agent-linux -id test-agent -relay-url wss://localhost:8443/ws/agent -allow 127.0.0.1:22"
echo "4. Run client: ./bin/client-linux -L :2222 -relay-url wss://localhost:8443/ws/client -agent test-agent -target 127.0.0.1:22"
