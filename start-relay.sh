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
    
    if [ -z "$RELAY_ADDR" ]; then
        export RELAY_ADDR=":8443"
        echo "⚠️  Warning: RELAY_ADDR not set, using default :8443"
    fi
    
    # Set certificate paths from environment or defaults
    if [ -z "$RELAY_CERT_FILE" ]; then
        export RELAY_CERT_FILE="certs/server.crt"
    fi
    
    if [ -z "$RELAY_KEY_FILE" ]; then
        export RELAY_KEY_FILE="certs/server.key"
    fi
    
else
    echo "❌ Error: .env.production not found!"
    echo "Please create .env.production file with required configuration"
    echo "Example:"
    echo "TUNNEL_TOKEN=your-secure-token"
    echo "RELAY_ADDR=:8443"
    echo "RELAY_HOST=sh.adisaputra.online"
    exit 1
fi

echo
echo "Production Configuration:"
echo "========================="
echo "- Relay Host: ${RELAY_HOST:-localhost}"
echo "- Listen Address: $RELAY_ADDR"
echo "- Token: ${TUNNEL_TOKEN:0:10}..." # Show only first 10 chars for security
echo "- Certificate: $RELAY_CERT_FILE"
echo "- Private Key: $RELAY_KEY_FILE"
echo "- TLS Enabled: ${TLS_ENABLED:-true}"
echo "========================="

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

# Create certificate directory if specified in env
CERT_DIR=$(dirname "$RELAY_CERT_FILE")
mkdir -p "$CERT_DIR"

# Generate self-signed certificates if they don't exist
if [ ! -f "$RELAY_CERT_FILE" ] || [ ! -f "$RELAY_KEY_FILE" ]; then
    echo "Certificates not found at $RELAY_CERT_FILE / $RELAY_KEY_FILE"
    echo "Generating self-signed certificates..."
    
    if [ -f "generate-certs.sh" ]; then
        ./generate-certs.sh
    else
        echo "Manual certificate generation for ${RELAY_HOST:-sh.adisaputra.online}..."
        openssl req -x509 -newkey rsa:2048 -keyout "$RELAY_KEY_FILE" -out "$RELAY_CERT_FILE" -days 365 -nodes \
            -subj "/C=ID/ST=Jakarta/L=Jakarta/O=RemoteTunnel/OU=IT/CN=${RELAY_HOST:-sh.adisaputra.online}" \
            -addext "subjectAltName=DNS:${RELAY_HOST:-sh.adisaputra.online},DNS:*.${RELAY_HOST:-sh.adisaputra.online},DNS:localhost,IP:127.0.0.1"
        chmod 600 "$RELAY_KEY_FILE"
        chmod 644 "$RELAY_CERT_FILE"
    fi
    echo "✅ Self-signed certificates generated"
fi

# Set certificate arguments
if [ -f "$RELAY_CERT_FILE" ] && [ -f "$RELAY_KEY_FILE" ]; then
    CERT_ARGS="-cert $RELAY_CERT_FILE -key $RELAY_KEY_FILE"
    echo "✅ Using certificates: $RELAY_CERT_FILE"
else
    CERT_ARGS=""
    echo "⚠️  No certificates found - relay will auto-generate basic ones"
fi

echo
echo "Compression options:"
echo "[1] No compression support (faster processing)"
echo "[2] Enable compression support (better bandwidth utilization)"
read -p "Select compression option (1-2) [1]: " compression_choice
compression_choice=${compression_choice:-1}

if [ "$compression_choice" = "2" ]; then
    COMPRESSION_FLAG="-compress"
    echo "Selected: Compression support enabled"
else
    COMPRESSION_FLAG=""
    echo "Selected: No compression support"
fi

echo
echo "🚀 Starting relay server..."
echo "Command: ./bin/relay -addr $RELAY_ADDR $CERT_ARGS -token $TUNNEL_TOKEN $COMPRESSION_FLAG"
echo
echo "📡 Endpoints:"
echo "- Agent: wss://${RELAY_HOST:-sh.adisaputra.online}:${RELAY_PORT:-8443}/ws/agent"
echo "- Client: wss://${RELAY_HOST:-sh.adisaputra.online}:${RELAY_PORT:-8443}/ws/client"
echo "- Health: https://${RELAY_HOST:-sh.adisaputra.online}:${RELAY_PORT:-8443}/health"
echo
echo "Press Ctrl+C to stop"
echo "========================================"

# Start relay server
./bin/relay -addr "$RELAY_ADDR" $CERT_ARGS -token "$TUNNEL_TOKEN" $COMPRESSION_FLAG
