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
    export $(grep -v '^#' .env.production | xargs)
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
mkdir -p /etc/ssl/tunnel 2>/dev/null || mkdir -p ./certs

# Set certificate paths
if [ -n "$RELAY_CERT_FILE" ] && [ -n "$RELAY_KEY_FILE" ]; then
    CERT_ARGS="-cert $RELAY_CERT_FILE -key $RELAY_KEY_FILE"
    echo "Using provided certificates"
else
    CERT_ARGS=""
    echo "Will auto-generate self-signed certificates"
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
