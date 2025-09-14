#!/bin/bash
# Build script for SSH Tunnel components

echo "Building SSH Tunnel components..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build Relay Server
echo "Building Relay Server..."
go build -o bin/tunnel-relay ./cmd/relay
if [ $? -ne 0 ]; then
    echo "Failed to build relay server"
    exit 1
fi
echo "Relay Server built successfully: bin/tunnel-relay"

# Build Agent
echo "Building Agent..."
go build -o bin/tunnel-agent ./cmd/agent
if [ $? -ne 0 ]; then
    echo "Failed to build agent"
    exit 1
fi
echo "Agent built successfully: bin/tunnel-agent"

# Build Client
echo "Building Client..."
go build -o bin/tunnel-client ./cmd/client
if [ $? -ne 0 ]; then
    echo "Failed to build client"
    exit 1
fi
echo "Client built successfully: bin/tunnel-client"

echo ""
echo "All components built successfully!"
echo ""
echo "Available executables:"
echo "  bin/tunnel-relay  - Relay server"
echo "  bin/tunnel-agent  - SSH agent"  
echo "  bin/tunnel-client - Tunnel client"
echo ""
echo "Run with -h flag to see usage options for each component."