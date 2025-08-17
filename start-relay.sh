#!/bin/bash
# Production setup script for relay server
# To be run on server 103.195.169.32

set -e

echo "========================================"
echo "Remote Tunnel - Relay Server Setup"
echo "========================================"
echo "Server Domain: sh.adisaputra.online"
echo "Listening on: Port 8443 (HTTPS/WSS)"
echo "========================================"

# Load configuration
if [ -f ".env.production" ]; then
    echo "Loading production configuration..."
    set -a  # automatically export all variables
    source .env.production
    set +a  # stop auto-export
    
    # Ensure RELAY_ADDR is set
    if [ -z "$RELAY_ADDR" ]; then
        export RELAY_ADDR=":8443"
        echo "Warning: RELAY_ADDR not set, using default :8443"
    fi
else
    echo "Warning: .env.production not found, using defaults"
    export TUNNEL_TOKEN="change-this-token"
    export RELAY_ADDR=":8443"
fi

echo
echo "Configuration:"
echo "- Token: $TUNNEL_TOKEN"
echo "- Listen Address: $RELAY_ADDR"
echo "- Certificate: ${RELAY_CERT_FILE:-auto-generated}"
echo

# Check if running as root (needed for port 8443)
if [ "$EUID" -ne 0 ]; then
    echo "Warning: Not running as root. Port 8443 may not be accessible."
    echo "Consider running with sudo or use a different port."
fi

# Check if binaries exist
if [ ! -f "bin/relay" ]; then
    echo "Error: relay binary not found. Building..."
    make build
    if [ $? -ne 0 ]; then
        echo "Build failed!"
        exit 1
    fi
fi

# Create certificate directory
mkdir -p ./certs

# Generate self-signed certificates if they don't exist
if [ ! -f "certs/server.crt" ] || [ ! -f "certs/server.key" ]; then
    echo "Self-signed certificates not found. Generating..."
    if [ -f "generate-certs.sh" ]; then
        ./generate-certs.sh
    else
        echo "Manual certificate generation..."
        openssl req -x509 -newkey rsa:2048 -keyout certs/server.key -out certs/server.crt -days 365 -nodes \
            -subj "/C=ID/ST=Jakarta/L=Jakarta/O=RemoteTunnel/OU=IT/CN=sh.adisaputra.online" \
            -addext "subjectAltName=DNS:sh.adisaputra.online,DNS:*.sh.adisaputra.online,DNS:localhost,IP:127.0.0.1"
        chmod 600 certs/server.key
        chmod 644 certs/server.crt
    fi
    echo "✅ Self-signed certificates generated"
fi

# Set certificate paths
if [ -f "certs/server.crt" ] && [ -f "certs/server.key" ]; then
    CERT_ARGS="-cert certs/server.crt -key certs/server.key"
    echo "Using self-signed certificates"
else
    CERT_ARGS=""
    echo "No certificates found - relay will auto-generate basic ones"
fi

echo
echo "Starting relay server..."
echo "Command: ./bin/relay -addr $RELAY_ADDR $CERT_ARGS -token $TUNNEL_TOKEN"
echo
echo "Endpoints:"
echo "- Agent: wss://sh.adisaputra.online:8443/ws/agent"
echo "- Client: wss://sh.adisaputra.online:8443/ws/client"
echo "- Health: https://sh.adisaputra.online:8443/health"
echo
echo "Press Ctrl+C to stop"
echo "========================================"

# Start relay server
./bin/relay -addr "$RELAY_ADDR" $CERT_ARGS -token "$TUNNEL_TOKEN"
