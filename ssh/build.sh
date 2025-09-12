#!/bin/bash
# Build script for Tunnel System (Linux/macOS)

echo "🚀 Building Tunnel System..."

# Create directories
mkdir -p bin logs

# Download dependencies
echo "📦 Downloading dependencies..."
go mod download
go mod tidy

# Build relay server
echo "🏗️ Building relay server..."
go build -o bin/tunnel-relay ./cmd/relay

# Build tunnel agent
echo "🏗️ Building tunnel agent..."
go build -o bin/tunnel-agent ./cmd/tunnel-agent

# Build tunnel client
echo "🏗️ Building tunnel client..."
go build -o bin/tunnel-client ./cmd/tunnel-client

echo "✅ Build complete!"
echo ""
echo "📂 Binaries created in bin/ directory:"
echo "  - tunnel-relay"
echo "  - tunnel-agent"  
echo "  - tunnel-client"
echo ""
echo "🚀 Usage Examples:"
echo ""
echo "1️⃣ Start Relay Server:"
echo "   ./bin/tunnel-relay -addr :8443"
echo ""
echo "2️⃣ Start Agent (on remote server behind NAT):"
echo "   ./bin/tunnel-agent -id my-agent -name \"My Server\" -relay-url ws://relay-server:8443/ws/agent"
echo ""
echo "3️⃣ Use Client (interactive mode):"
echo "   ./bin/tunnel-client -relay-url ws://relay-server:8443/ws/client -i"
echo ""
echo "4️⃣ Use Client (direct tunnel):"
echo "   ./bin/tunnel-client -L 2222 -agent my-agent -target 127.0.0.1:22 -relay-url ws://relay-server:8443/ws/client"
echo ""
echo "📋 Common Targets:"
echo "   SSH:        127.0.0.1:22"
echo "   MySQL:      127.0.0.1:3306"
echo "   PostgreSQL: 127.0.0.1:5432"
echo ""
echo "📊 Monitoring:"
echo "   Health:     http://relay-server:8443/health"
echo "   Agents:     http://relay-server:8443/api/agents"
echo "   Tunnels:    http://relay-server:8443/api/tunnels"
