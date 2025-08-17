#!/bin/bash
# Quick setup script for sh.adisaputra.online deployment

echo "========================================"
echo "Remote Tunnel - Quick Setup for Domain"
echo "========================================"

DOMAIN="sh.adisaputra.online"

echo "Setting up remote tunnel for $DOMAIN..."
echo

echo "[1/6] Building binaries..."
if ./build.sh; then
    echo "✅ Build successful"
else
    echo "❌ Build failed!"
    exit 1
fi

echo
echo "[2/6] Testing domain connectivity..."
./test-domain.sh

echo
echo "[3/6] Generating secure token..."
./generate-token.sh

echo
echo "[4/6] Configuring environment for $DOMAIN..."
if [ ! -f ".env.production" ]; then
    echo "Creating .env.production..."
    cat > .env.production << EOF
RELAY_HOST=$DOMAIN
RELAY_PORT=8443
RELAY_URL=wss://$DOMAIN:8443/ws
TLS_ENABLED=true
CERT_FILE=certs/server.crt
KEY_FILE=certs/server.key
TOKEN=your-secure-token-here
LOG_LEVEL=info
EOF
    echo
    echo "⚠️  IMPORTANT: Edit .env.production and set your secure token!"
else
    echo "✅ .env.production already exists"
fi

echo
echo "[5/6] Setting up certificates directory..."
mkdir -p certs
if [ ! -f "certs/server.crt" ] || [ ! -f "certs/server.key" ]; then
    echo "Generating self-signed certificates..."
    if [ -f "generate-certs.sh" ]; then
        ./generate-certs.sh
    else
        echo "Manual certificate generation..."
        openssl req -x509 -newkey rsa:2048 -keyout certs/server.key -out certs/server.crt -days 365 -nodes \
            -subj "/C=ID/ST=Jakarta/L=Jakarta/O=RemoteTunnel/OU=IT/CN=$DOMAIN"
        chmod 600 certs/server.key
        chmod 644 certs/server.crt
    fi
    echo "✅ Self-signed certificates generated"
else
    echo "✅ Certificates already exist"
fi
echo "✅ Certificates directory ready"

echo
echo "[6/6] Creating agent configuration..."
echo "Creating config/agent.yaml..."
mkdir -p config

cat > config/agent.yaml << EOF
# Agent Configuration for $DOMAIN
agent_id: "laptop-agent"
relay_url: "wss://$DOMAIN:8443/ws/agent"
token: "your-secure-token-here"
log_level: "info"
services:
  - name: "ssh"
    target: "127.0.0.1:22"
  - name: "rdp"
    target: "127.0.0.1:3389"
  - name: "web"
    target: "127.0.0.1:8080"
EOF

echo "✅ Agent configuration created"

echo
echo "========================================"
echo "Setup Complete!"
echo "========================================"
echo
echo "Next steps:"
echo
echo "1. Edit .env.production and set your secure token"
echo "2. Edit config/agent.yaml and set the same token"
echo "3. Deploy relay to $DOMAIN:"
echo "   deploy/deploy-domain.sh (on server)"
echo "4. Start agent on this laptop:"
echo "   ./start-agent.sh"
echo "5. Connect from remote machine:"
echo "   ./bin/client -L :2222 -relay-url wss://$DOMAIN:8443/ws/client -agent laptop-agent -target 127.0.0.1:22 -token YOUR_TOKEN"
echo
echo "Security Notes:"
echo "- Use a strong, unique token (minimum 32 characters)"
echo "- Configure TLS certificates properly on the relay server"
echo "- Restrict agent services to only what you need"
echo "- Monitor logs for suspicious activity"
echo
echo "For detailed instructions, see:"
echo "- QUICKSTART.md"
echo "- DEPLOYMENT.md"
echo

read -p "Press Enter to continue..."
