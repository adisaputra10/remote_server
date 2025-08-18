#!/bin/bash
# Production setup script for Linux laptop (Agent)
# Relay Server: 103.195.169.32

set -e

echo "========================================"
echo "Remote Tunnel - Laptop Agent Setup"
echo "========================================"
echo "Relay Server: sh.adisaputra.online"
echo "Agent: Local Laptop"
echo "========================================"

# Load configuration from .env.production
if [ -f ".env.production" ]; then
    echo "Loading production configuration from .env.production..."
    
    # Load environment variables, ignoring comments and empty lines
    set -a  # automatically export all variables
    source <(grep -v '^#\|^$' .env.production)
    set +a  # stop auto-export
    
    echo "✅ Production configuration loaded"
    
    # Validate required variables
    if [ -z "$TUNNEL_TOKEN" ]; then
        echo "❌ Error: TUNNEL_TOKEN not set in .env.production"
        exit 1
    fi
    
    if [ -z "$AGENT_ID" ]; then
        export AGENT_ID="laptop-agent"
        echo "⚠️  Warning: AGENT_ID not set, using default: laptop-agent"
    fi
    
    if [ -z "$AGENT_RELAY_URL" ]; then
        export AGENT_RELAY_URL="wss://${RELAY_HOST:-sh.adisaputra.online}:${RELAY_PORT:-8443}/ws/agent"
        echo "⚠️  Warning: AGENT_RELAY_URL not set, using default: $AGENT_RELAY_URL"
    fi
    
else
    echo "❌ Error: .env.production not found!"
    echo "Please create .env.production file with required configuration"
    echo "Example:"
    echo "TUNNEL_TOKEN=your-secure-token"
    echo "AGENT_ID=laptop-agent"
    echo "AGENT_RELAY_URL=wss://sh.adisaputra.online:8443/ws/agent"
    exit 1
fi

echo
echo "Production Configuration:"
echo "========================="
echo "- Agent ID: $AGENT_ID"
echo "- Relay URL: $AGENT_RELAY_URL"
echo "- Token: ${TUNNEL_TOKEN:0:10}..." # Show only first 10 chars for security
echo "- Relay Host: ${RELAY_HOST:-sh.adisaputra.online}"
echo "========================="

# Check if binaries exist
AGENT_BINARY=""
if [ -f "bin/agent" ]; then
    AGENT_BINARY="bin/agent"
elif [ -f "bin/agent-linux" ]; then
    AGENT_BINARY="bin/agent-linux"
else
    echo "Error: agent binary not found. Building..."
    make build
    if [ $? -ne 0 ]; then
        echo "Build failed!"
        exit 1
    fi
    AGENT_BINARY="bin/agent"
fi

echo "Using binary: $AGENT_BINARY"
echo

echo "Services to expose:"
echo "[1] SSH Server (port 22) - from .env.production"
echo "[2] Web Services (ports 80,443,8080,3000) - from .env.production"
echo "[3] Database Services (ports 5432,3306,6379) - from .env.production"
echo "[4] All configured services from .env.production"
echo "[5] Custom ports (manual input)"
read -p "Select option (1-5): " choice

case $choice in
    1)
        if [ -n "$AGENT_ALLOW_SSH" ]; then
            ALLOW_PORTS="-allow $AGENT_ALLOW_SSH"
            echo "Selected: SSH Server ($AGENT_ALLOW_SSH)"
        else
            ALLOW_PORTS="-allow 127.0.0.1:22"
            echo "Selected: SSH Server (127.0.0.1:22 - default)"
        fi
        ;;
    2)
        WEB_PORTS=""
        [ -n "$AGENT_ALLOW_HTTP" ] && WEB_PORTS="$WEB_PORTS -allow $AGENT_ALLOW_HTTP"
        [ -n "$AGENT_ALLOW_HTTPS" ] && WEB_PORTS="$WEB_PORTS -allow $AGENT_ALLOW_HTTPS"
        [ -n "$AGENT_ALLOW_WEB" ] && WEB_PORTS="$WEB_PORTS -allow $AGENT_ALLOW_WEB"
        [ -n "$AGENT_ALLOW_DEV" ] && WEB_PORTS="$WEB_PORTS -allow $AGENT_ALLOW_DEV"
        
        if [ -n "$WEB_PORTS" ]; then
            ALLOW_PORTS="$WEB_PORTS"
            echo "Selected: Web Services from .env.production"
        else
            ALLOW_PORTS="-allow 127.0.0.1:80 -allow 127.0.0.1:443 -allow 127.0.0.1:8080 -allow 127.0.0.1:3000"
            echo "Selected: Web Services (default ports)"
        fi
        ;;
    3)
        DB_PORTS=""
        [ -n "$AGENT_ALLOW_POSTGRES" ] && DB_PORTS="$DB_PORTS -allow $AGENT_ALLOW_POSTGRES"
        [ -n "$AGENT_ALLOW_MYSQL" ] && DB_PORTS="$DB_PORTS -allow $AGENT_ALLOW_MYSQL"
        [ -n "$AGENT_ALLOW_REDIS" ] && DB_PORTS="$DB_PORTS -allow $AGENT_ALLOW_REDIS"
        
        if [ -n "$DB_PORTS" ]; then
            ALLOW_PORTS="$DB_PORTS"
            echo "Selected: Database Services from .env.production"
        else
            ALLOW_PORTS="-allow 127.0.0.1:5432 -allow 127.0.0.1:3306 -allow 127.0.0.1:6379"
            echo "Selected: Database Services (default ports)"
        fi
        ;;
    4)
        ALL_PORTS=""
        [ -n "$AGENT_ALLOW_SSH" ] && ALL_PORTS="$ALL_PORTS -allow $AGENT_ALLOW_SSH"
        [ -n "$AGENT_ALLOW_HTTP" ] && ALL_PORTS="$ALL_PORTS -allow $AGENT_ALLOW_HTTP"
        [ -n "$AGENT_ALLOW_HTTPS" ] && ALL_PORTS="$ALL_PORTS -allow $AGENT_ALLOW_HTTPS"
        [ -n "$AGENT_ALLOW_WEB" ] && ALL_PORTS="$ALL_PORTS -allow $AGENT_ALLOW_WEB"
        [ -n "$AGENT_ALLOW_DEV" ] && ALL_PORTS="$ALL_PORTS -allow $AGENT_ALLOW_DEV"
        [ -n "$AGENT_ALLOW_POSTGRES" ] && ALL_PORTS="$ALL_PORTS -allow $AGENT_ALLOW_POSTGRES"
        [ -n "$AGENT_ALLOW_MYSQL" ] && ALL_PORTS="$ALL_PORTS -allow $AGENT_ALLOW_MYSQL"
        [ -n "$AGENT_ALLOW_REDIS" ] && ALL_PORTS="$ALL_PORTS -allow $AGENT_ALLOW_REDIS"
        
        if [ -n "$ALL_PORTS" ]; then
            ALLOW_PORTS="$ALL_PORTS"
            echo "Selected: All services from .env.production"
        else
            ALLOW_PORTS="-allow 127.0.0.1:22 -allow 127.0.0.1:80 -allow 127.0.0.1:443 -allow 127.0.0.1:3000 -allow 127.0.0.1:8080 -allow 127.0.0.1:5432 -allow 127.0.0.1:3306"
            echo "Selected: All common services (defaults)"
        fi
        ;;
    5)
        read -p "Enter ports (e.g., 127.0.0.1:3000 127.0.0.1:8000): " custom_ports
        ALLOW_PORTS="-allow ${custom_ports// / -allow }"
        echo "Selected: Custom ports"
        ;;
    *)
        echo "Invalid option, using SSH only"
        ALLOW_PORTS="-allow ${AGENT_ALLOW_SSH:-127.0.0.1:22}"
        ;;
esac

echo
echo "Compression options:"
echo "[1] No compression (faster for local networks)"
echo "[2] Enable gzip compression (slower but saves bandwidth)"
read -p "Select compression option (1-2) [1]: " compression_choice
compression_choice=${compression_choice:-1}

if [ "$compression_choice" = "2" ]; then
    COMPRESSION_FLAG="-compress"
    echo "Selected: Gzip compression enabled"
else
    COMPRESSION_FLAG=""
    echo "Selected: No compression"
fi

echo
echo "🚀 Starting agent with configuration:"
echo "Agent ID: $AGENT_ID"
echo "Relay URL: $AGENT_RELAY_URL"
echo "Allowed Ports: $ALLOW_PORTS"
echo "Compression: ${COMPRESSION_FLAG:-disabled}"
echo
echo "Command: $AGENT_BINARY -id $AGENT_ID -relay-url $AGENT_RELAY_URL $ALLOW_PORTS -token $TUNNEL_TOKEN -insecure $COMPRESSION_FLAG"
echo
echo "Press Ctrl+C to stop the agent"
echo "========================================"

# Start agent with insecure flag for self-signed certificates
exec $AGENT_BINARY -id "$AGENT_ID" -relay-url "$AGENT_RELAY_URL" $ALLOW_PORTS -token "$TUNNEL_TOKEN" -insecure $COMPRESSION_FLAG
